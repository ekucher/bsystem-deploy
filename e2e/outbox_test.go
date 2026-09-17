package e2e

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Metric reads one sample from the platform's exposition.
//
// The scenarios below assert on the outbox through /metrics rather than by
// subscribing to the bus, and that is deliberate. A core NATS subscriber only
// receives what is published while it is subscribed, so a test that stops the
// broker, produces an event and subscribes again is racing the publisher's
// next pass — and a race that usually wins is a test that occasionally fails
// for a reason nobody can reproduce. The queue depth is the same property
// stated in a form that holds still.
func (h *Harness) Metric(base, name string, labels map[string]string) (float64, bool, error) {
	response, err := h.Do(http.MethodGet, base+"/metrics", "", nil)
	if err != nil {
		return 0, false, err
	}
	if response.Status != http.StatusOK {
		return 0, false, fmt.Errorf("GET /metrics: status = %d", response.Status)
	}
	for _, line := range strings.Split(string(response.Body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.HasPrefix(line, name) {
			continue
		}
		rest := line[len(name):]
		if !strings.HasPrefix(rest, "{") && !strings.HasPrefix(rest, " ") {
			// A different family whose name starts with this one.
			continue
		}
		if !matchesLabels(rest, labels) {
			continue
		}
		fields := strings.Fields(line)
		value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			return 0, false, fmt.Errorf("parse %s: %w", line, err)
		}
		return value, true, nil
	}
	return 0, false, nil
}

func matchesLabels(rest string, labels map[string]string) bool {
	for name, value := range labels {
		if !strings.Contains(rest, name+"=\""+value+"\"") {
			return false
		}
	}
	return true
}

func (h *Harness) metricOrZero(t *testing.T, base, name string, labels map[string]string) float64 {
	t.Helper()
	value, _, err := h.Metric(base, name, labels)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return value
}

// The claim the outbox exists for, executed against a real broker outage.
//
// Before P29 this event was published from the request path with no
// acknowledgement, so an allocation made while NATS was gone was announced to
// nobody and there was no second chance at it: a Global ID is minted once per
// source record and no later event restates it.
//
// The scenario mints one while the broker is stopped and then watches the
// queue drain when it returns.
func TestAGlobalIDMintedWhileTheBrokerIsDownIsAnnouncedWhenItReturns(t *testing.T) {
	harness := ready(t)
	control := LifecycleControl(t)

	const queued = "bsystem_event_outbox_events"
	const attempts = "bsystem_event_outbox_attempts_total"
	deliveredLabels := map[string]string{"subject": "bsystem.events.global_id.created", "outcome": "delivered"}
	deliveredBefore := harness.metricOrZero(t, harness.Core, attempts, deliveredLabels)

	control.Stop(t, ServiceNATS)

	// The allocation itself must not be affected by the broker being gone.
	// Refusing a change because its announcement cannot be delivered would
	// lose the change to make a secondary effect look atomic.
	sourceID := fmt.Sprintf("e2e-outbox-%d", time.Now().UnixNano())
	created := harness.API(t, http.MethodPost, "/api/v1/global-ids", TokenAdmin,
		map[string]any{"entity_type": "client", "source": "e2e", "source_id": sourceID})
	if created.Status != http.StatusCreated {
		t.Fatalf("allocating a Global ID while NATS is gone: status = %d, want 201 (body: %s)", created.Status, truncate(created.Body))
	}
	var entity struct {
		GlobalID string `json:"global_id"`
	}
	created.JSON(t, &entity)
	if !strings.HasPrefix(entity.GlobalID, "CL-") {
		t.Fatalf("global_id = %q, want a CL-* identifier", entity.GlobalID)
	}

	// Queued rather than dropped. This is the assertion that fails against
	// the code as it stood before P29: a fire-and-forget publish to an absent
	// broker leaves nothing behind at all.
	waitFor(t, "the announcement to be queued", 30*time.Second, func() (bool, string) {
		depth := harness.metricOrZero(t, harness.Core, queued, map[string]string{"state": "queued"})
		return depth > 0, fmt.Sprintf("queued = %v", depth)
	})

	control.Start(t, ServiceNATS)

	// And delivered once the broker is back, with no restart and no
	// intervention. The queue draining is one half; an acknowledged delivery
	// counted against this subject is the other, because a row could also
	// leave the queue by exhausting its retries.
	waitFor(t, "the queue to drain", 90*time.Second, func() (bool, string) {
		depth := harness.metricOrZero(t, harness.Core, queued, map[string]string{"state": "queued"})
		return depth == 0, fmt.Sprintf("queued = %v", depth)
	})
	waitFor(t, "the announcement to be acknowledged by the broker", 60*time.Second, func() (bool, string) {
		now := harness.metricOrZero(t, harness.Core, attempts, deliveredLabels)
		return now > deliveredBefore, fmt.Sprintf("delivered = %v, was %v", now, deliveredBefore)
	})

	// Nothing may be left permanently undeliverable by a two-minute outage.
	if failed := harness.metricOrZero(t, harness.Core, queued, map[string]string{"state": "failed"}); failed > 0 {
		t.Errorf("%v events are permanently failed after a survivable outage", failed)
	}
}

// The other half of the contract: asking twice for the same source record is
// one allocation and must be one announcement. The handler used to publish
// after every call, so a consumer counting allocations counted requests.
func TestOneSourceRecordIsAnnouncedOnceHoweverOftenItIsRequested(t *testing.T) {
	harness := ready(t)

	const attempts = "bsystem_event_outbox_attempts_total"
	labels := map[string]string{"subject": "bsystem.events.global_id.created", "outcome": "delivered"}

	// Start from a drained queue. The counter is platform-wide, so a straggler
	// queued by an earlier scenario and delivered inside this one's window
	// would be counted as a second announcement of this source record — a test
	// that fails for something another test did.
	waitFor(t, "the outbox to drain before counting", 60*time.Second, func() (bool, string) {
		depth := harness.metricOrZero(t, harness.Core, "bsystem_event_outbox_events", map[string]string{"state": "queued"})
		return depth == 0, fmt.Sprintf("queued = %v", depth)
	})
	before := harness.metricOrZero(t, harness.Core, attempts, labels)

	sourceID := fmt.Sprintf("e2e-once-%d", time.Now().UnixNano())
	body := map[string]any{"entity_type": "client", "source": "e2e", "source_id": sourceID}
	first := harness.API(t, http.MethodPost, "/api/v1/global-ids", TokenAdmin, body)
	if first.Status != http.StatusCreated {
		t.Fatalf("first allocation: status = %d (body: %s)", first.Status, truncate(first.Body))
	}
	for i := 0; i < 3; i++ {
		again := harness.API(t, http.MethodPost, "/api/v1/global-ids", TokenAdmin, body)
		if again.Status != http.StatusCreated && again.Status != http.StatusOK {
			t.Fatalf("repeat allocation %d: status = %d (body: %s)", i, again.Status, truncate(again.Body))
		}
	}

	// Exactly one delivery, not four. The wait is for the publisher's pass;
	// the assertion is on the count.
	waitFor(t, "the single announcement to be delivered", 60*time.Second, func() (bool, string) {
		now := harness.metricOrZero(t, harness.Core, attempts, labels)
		return now >= before+1, fmt.Sprintf("delivered = %v, was %v", now, before)
	})
	// Give any further announcement a pass or two to appear before counting.
	time.Sleep(6 * time.Second)
	after := harness.metricOrZero(t, harness.Core, attempts, labels)
	if after != before+1 {
		t.Errorf("four allocation calls for one source record produced %v deliveries, want 1", after-before)
	}
}
