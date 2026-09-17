package e2e

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// This file exists because of how the suite skips.
//
// Every scenario calls New, and New skips when E2E_BASE_URL is empty, so that
// `go test ./...` stays safe on a machine with no stack up. That is the right
// behaviour locally and the wrong one in CI: a skipped Go test exits zero, so
// if the variable is ever missing where the suite is meant to run — a renamed
// variable, a restructured job, an env block edited a step too high — every
// scenario skips, the job succeeds, and the "Autonomous E2E" check goes green
// having verified nothing.
//
// That failure is silent by construction and would not be noticed until
// something the E2E stack was supposed to catch reached a deployment. So the
// caller states which it wants: E2E_REQUIRED turns a missing stack from a skip
// into a failure.

// ErrStackRequired is returned when the suite was told the stack is mandatory
// and no stack is configured.
var ErrStackRequired = errors.New("E2E_REQUIRED is set but E2E_BASE_URL is empty")

// checkRequired reports whether the suite may skip for want of a stack.
//
// It takes its inputs rather than reading the environment so that the guard
// itself is testable — a guard nothing exercises is exactly the kind of thing
// it exists to prevent.
func checkRequired(baseURL, required string) error {
	if strings.TrimSpace(required) == "" {
		return nil
	}
	if strings.TrimSpace(baseURL) != "" {
		return nil
	}
	return fmt.Errorf("%w: the stack is mandatory here, and a skipped suite would report success without running a single scenario", ErrStackRequired)
}

// TestMain refuses to run at all when the stack is required and absent, rather
// than letting each scenario skip on its own. One check in one place cannot be
// bypassed by a new scenario that forgets it.
//
// It lives in a _test.go file because that is the only place the testing
// framework looks for it. A TestMain in an ordinary file compiles, reads
// correctly and never runs — which would have left this guard as decoration
// over the very failure it exists to prevent.
func TestMain(m *testing.M) {
	if err := checkRequired(os.Getenv("E2E_BASE_URL"), os.Getenv("E2E_REQUIRED")); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: %v\n", err)
		os.Exit(1)
	}
	// The same refusal one level down: a required run whose lifecycle
	// scenarios cannot stop a dependency would skip every outage scenario and
	// still exit zero. See lifecycle.go.
	if err := checkLifecycleRequired(os.Getenv("E2E_LIFECYCLE"), os.Getenv("E2E_REQUIRED")); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// The guard decides whether a missing stack is a skip or a failure, and it is
// the only thing standing between a renamed environment variable and a green
// check that ran nothing. So it is covered from both sides, including the two
// cases that look like the answer but are not: a variable set to whitespace,
// which a shell produces easily, and a required run that is properly
// configured, which must not be turned into a failure.
func TestTheStackRequirementGuard(t *testing.T) {
	stack := "http://127.0.0.1:8080"

	t.Run("a required run with no stack fails rather than skipping", func(t *testing.T) {
		err := checkRequired("", "1")
		if err == nil {
			t.Fatal("a required run with no stack was allowed to skip; the suite would report success having run nothing")
		}
		if !errors.Is(err, ErrStackRequired) {
			t.Errorf("error does not identify itself: %v", err)
		}
	})

	t.Run("a required run with a stack proceeds", func(t *testing.T) {
		if err := checkRequired(stack, "1"); err != nil {
			t.Errorf("a configured required run was refused: %v", err)
		}
	})

	t.Run("an unrequired run with no stack may skip", func(t *testing.T) {
		// The local case: `go test ./...` on a machine with no stack up must
		// stay safe, which is why New skips at all.
		if err := checkRequired("", ""); err != nil {
			t.Errorf("a local run without a stack was refused: %v", err)
		}
	})

	t.Run("an unrequired run with a stack proceeds", func(t *testing.T) {
		if err := checkRequired(stack, ""); err != nil {
			t.Errorf("a local run with a stack was refused: %v", err)
		}
	})

	// A variable a shell set to nothing useful. `E2E_REQUIRED=" "` reads as
	// set to a naive check and means nothing to a person, so it is treated as
	// unset rather than silently enabling a mode nobody asked for. The mirror
	// case matters more: a whitespace E2E_BASE_URL must not satisfy a required
	// run, since the suite could not reach a stack at that address either.
	t.Run("whitespace is not a value", func(t *testing.T) {
		if err := checkRequired("", "   "); err != nil {
			t.Errorf("a whitespace E2E_REQUIRED enabled the requirement: %v", err)
		}
		if err := checkRequired("   ", "1"); err == nil {
			t.Error("a whitespace E2E_BASE_URL satisfied a required run; no stack is reachable there")
		}
	})

	// The message a person reads when CI fails this way has to say what
	// happened, because the symptom — a suite that refuses to start — looks
	// nothing like the cause.
	t.Run("says why it refused", func(t *testing.T) {
		err := checkRequired("", "1")
		if err == nil {
			t.Fatal("no error")
		}
		for _, mention := range []string{"E2E_REQUIRED", "E2E_BASE_URL"} {
			if !strings.Contains(err.Error(), mention) {
				t.Errorf("the refusal does not name %s: %v", mention, err)
			}
		}
	})
}
