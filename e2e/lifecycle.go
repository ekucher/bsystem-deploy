package e2e

// Dependency lifecycle control for the E2E stack.
//
// Every other scenario in this package observes a stack that is fully up. That
// leaves the questions worth asking about a dependency unanswerable: what the
// platform does while NATS is gone, what it says while PostgreSQL is
// unreachable, and whether either recovers on its own once the dependency
// returns. Those were written down as expected behaviour in
// docs/STAGE-ACCEPTANCE.md and nothing executed them.
//
// The control surface is deliberately outside the stack. The test binary runs
// on the CI runner, next to the Docker daemon, and drives `docker compose`
// itself. No container is given the Docker socket, no application container
// gains a capability, and the compose files are unchanged by any of this — a
// scenario that could restart its own neighbours from inside the network would
// be a hole in the very isolation the rest of the stack is built to have.
//
// What may be controlled is an allowlist, not an argument. A helper that takes
// any service name lets a future scenario stop the Integration Core the whole
// suite is asserting against, and the failure that produces looks like a
// platform defect rather than a test that reached too far.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The services a lifecycle scenario may stop and start.
const (
	// ServiceNATS is the event bus. Documented as degraded-but-alive: its
	// absence must not make the platform unready.
	ServiceNATS = "nats"
	// ServicePostgres is the platform database. Documented as fatal to
	// readiness: its absence must make the platform unready and say so
	// without describing itself.
	ServicePostgres = "postgres"
	// ServicePartialCore is the second Integration Core, the one with Outline
	// unconfigured. It is on this list only so a scenario can prove what a
	// cold start against an unavailable database does, which cannot be
	// observed without restarting a core.
	//
	// The primary Integration Core is deliberately absent from the list. Every
	// other scenario in the package talks to it, and a restart of it from here
	// would surface as an unrelated failure somewhere else in the suite.
	ServicePartialCore = "integration-core-no-outline"
)

// controllable maps each permitted service to why it is permitted, so the
// reason travels with the permission instead of living in a commit message.
var controllable = map[string]string{
	ServiceNATS:        "the event bus, whose absence is documented as degraded rather than unready",
	ServicePostgres:    "the platform database, whose absence is documented as unready",
	ServicePartialCore: "the spare Integration Core, restarted to observe a cold start",
}

// checkControllable reports whether a service may be stopped from a scenario.
//
// It is a plain function over a string so that the allowlist is testable
// without Docker: the guard runs on every machine that runs `go test ./...`,
// not only on a runner that happens to have the stack up.
func checkControllable(service string) error {
	if _, ok := controllable[strings.TrimSpace(service)]; ok {
		return nil
	}
	permitted := make([]string, 0, len(controllable))
	for name := range controllable {
		permitted = append(permitted, name)
	}
	return fmt.Errorf("service %q may not be controlled by a scenario; the permitted services are %s", service, strings.Join(sorted(permitted), ", "))
}

func sorted(values []string) []string {
	out := append([]string(nil), values...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// ErrLifecycleRequired is returned when the suite is mandatory and the
// lifecycle scenarios were not enabled.
//
// The skip this prevents is the same one required_test.go exists for, one
// level down. A lifecycle scenario that cannot reach Docker skips, a skipped
// Go test exits zero, and the outage behaviour nobody tested is the outage
// behaviour a deployment meets first.
var ErrLifecycleRequired = fmt.Errorf("E2E_REQUIRED is set but E2E_LIFECYCLE is empty")

// checkLifecycleRequired reports whether the lifecycle scenarios may skip.
func checkLifecycleRequired(lifecycle, required string) error {
	if strings.TrimSpace(required) == "" {
		return nil
	}
	if strings.TrimSpace(lifecycle) != "" {
		return nil
	}
	return fmt.Errorf("%w: the dependency-outage scenarios would skip silently and the run would report success without ever stopping a dependency", ErrLifecycleRequired)
}

// Lifecycle drives the stack's dependency containers.
type Lifecycle struct {
	composeFile string
}

// LifecycleControl returns a controller, skipping the test when the run was
// not told it may stop containers.
func LifecycleControl(t *testing.T) *Lifecycle {
	t.Helper()
	if strings.TrimSpace(envRaw("E2E_LIFECYCLE")) == "" {
		t.Skip("E2E_LIFECYCLE is not set; these scenarios stop and start stack containers, so they are opt-in")
	}
	file := env("E2E_COMPOSE_FILE", "../docker-compose.e2e.yml")
	absolute, err := filepath.Abs(file)
	if err != nil {
		t.Fatalf("resolve %s: %v", file, err)
	}
	return &Lifecycle{composeFile: absolute}
}

func envRaw(name string) string {
	return env(name, "")
}

// compose runs one docker compose command against the E2E stack.
//
// The timeout is the point: a scenario that hangs on a daemon that never
// answers is indistinguishable from a platform that never recovers, and the
// first is a CI job that runs until the runner is reclaimed.
func (l *Lifecycle) compose(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	full := append([]string{"compose", "-f", l.composeFile}, args...)
	command := exec.CommandContext(ctx, "docker", full...)
	command.Dir = filepath.Dir(l.composeFile)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker %s: %w (output: %s)", strings.Join(full, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// Stop stops a dependency and restores it when the test ends.
//
// The cleanup is not a convenience. A scenario that stops PostgreSQL and then
// fails an assertion would otherwise leave every later scenario in the package
// talking to a platform whose database is gone, and the resulting wall of
// failures hides the one that is real.
func (l *Lifecycle) Stop(t *testing.T, service string) {
	t.Helper()
	if err := checkControllable(service); err != nil {
		t.Fatalf("%v", err)
	}
	if _, err := l.compose("stop", "--timeout", "10", service); err != nil {
		t.Fatalf("stop %s: %v", service, err)
	}
	t.Cleanup(func() {
		if _, err := l.compose("start", service); err != nil {
			t.Errorf("restoring %s after the scenario failed: %v", service, err)
		}
	})
}

// Start starts a stopped dependency.
func (l *Lifecycle) Start(t *testing.T, service string) {
	t.Helper()
	if err := checkControllable(service); err != nil {
		t.Fatalf("%v", err)
	}
	if _, err := l.compose("start", service); err != nil {
		t.Fatalf("start %s: %v", service, err)
	}
}

// Restart restarts a service in place.
func (l *Lifecycle) Restart(t *testing.T, service string) {
	t.Helper()
	if err := checkControllable(service); err != nil {
		t.Fatalf("%v", err)
	}
	if _, err := l.compose("restart", "--timeout", "10", service); err != nil {
		t.Fatalf("restart %s: %v", service, err)
	}
}

// Exec runs a command inside a stack container and returns its stdout.
//
// It is how the backup scenario reaches pg_dump and psql. The database is on
// an internal network and is not published, deliberately — the stack's own
// isolation is worth keeping, and a scenario that needed the port opened would
// have weakened the thing it is testing against.
//
// Stdin is closed (-T), because a compose exec attached to a terminal that is
// not there hangs rather than failing.
func (l *Lifecycle) Exec(t *testing.T, service string, args ...string) string {
	t.Helper()
	if err := checkControllable(service); err != nil {
		t.Fatalf("%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	full := append([]string{"compose", "-f", l.composeFile, "exec", "-T", service}, args...)
	command := exec.CommandContext(ctx, "docker", full...)
	command.Dir = filepath.Dir(l.composeFile)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("exec in %s: %v (stderr: %s)", service, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String()
}

// ExecInput runs a command inside a stack container with input on stdin.
func (l *Lifecycle) ExecInput(t *testing.T, service, input string, args ...string) string {
	t.Helper()
	if err := checkControllable(service); err != nil {
		t.Fatalf("%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	full := append([]string{"compose", "-f", l.composeFile, "exec", "-T", service}, args...)
	command := exec.CommandContext(ctx, "docker", full...)
	command.Dir = filepath.Dir(l.composeFile)
	command.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("exec in %s: %v (stderr: %s)", service, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String()
}

// StartedAt reports when a service's container last started.
//
// It is how "recovered without a restart" is proved rather than assumed. A
// core that crashed and was brought back by the restart policy also ends up
// answering /readyz, and from the outside the two are the same HTTP 200.
func (l *Lifecycle) StartedAt(t *testing.T, service string) string {
	t.Helper()
	id, err := l.compose("ps", "--quiet", service)
	if err != nil {
		t.Fatalf("locate the container for %s: %v", service, err)
	}
	container := strings.TrimSpace(id)
	if container == "" {
		t.Fatalf("no container is running for %s", service)
	}
	// compose ps --quiet prints one id per replica; the stack runs one.
	if index := strings.IndexByte(container, '\n'); index >= 0 {
		container = container[:index]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.StartedAt}}|{{.RestartCount}}", container).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect %s: %v (output: %s)", service, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}

// waitFor polls until condition holds, and says what it last saw when it does
// not. A bare "timed out" turns a real regression into a re-run.
func waitFor(t *testing.T, what string, timeout time.Duration, condition func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	last := "nothing observed yet"
	for {
		ok, observed := condition()
		if ok {
			return
		}
		last = observed
		if time.Now().After(deadline) {
			t.Fatalf("%s did not happen within %s; last observation: %s", what, timeout, last)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// readiness reads /readyz from a core and decodes it, without failing the test
// when the core is not answering at all: during an outage scenario a refused
// connection is an observation, not an error.
type readinessReport struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
	code   int
	raw    string
}

func (h *Harness) readiness(base string) (readinessReport, error) {
	response, err := h.Do("GET", base+"/readyz", "", nil)
	if err != nil {
		return readinessReport{}, err
	}
	report := readinessReport{code: response.Status, raw: string(response.Body)}
	if err := json.Unmarshal(response.Body, &report); err != nil {
		return report, fmt.Errorf("decode /readyz (HTTP %d): %w (body: %s)", response.Status, err, truncate(response.Body))
	}
	return report, nil
}

func (r readinessReport) String() string {
	return "HTTP " + strconv.Itoa(r.code) + " " + r.raw
}
