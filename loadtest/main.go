// Command loadtest drives the E2E stack concurrently and reports latency
// percentiles.
//
// It is a regression harness, not a capacity test. It runs against four mock
// upstreams on whatever machine is to hand, so its absolute numbers describe
// the mocks and the machine rather than the platform. What it is good for is
// the comparison: run it before a change and after one, and a difference
// means something.
//
//	go run ./loadtest -url http://127.0.0.1:8080 -concurrency 16 -requests 2000
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// target is one endpoint to exercise.
type target struct {
	Name  string
	Path  string
	Token string
}

func main() {
	base := flag.String("url", envOr("E2E_BASE_URL", "http://127.0.0.1:8080"), "Integration Core base URL")
	concurrency := flag.Int("concurrency", 16, "requests in flight at once")
	requests := flag.Int("requests", 1000, "total requests per target")
	timeout := flag.Duration("timeout", 30*time.Second, "per-request timeout")
	flag.Parse()

	// Test-only tokens served by the identity mock. They cannot reach any
	// real system.
	const (
		admin  = "test-token-admin"
		devops = "test-token-devops"
	)

	targets := []target{
		// Unauthenticated, so this measures the platform without authentik in
		// the path — the floor everything else is built on.
		{Name: "health", Path: "/health"},
		{Name: "readyz", Path: "/readyz"},
		// The identity path: every authenticated request pays for this.
		{Name: "me", Path: "/api/v1/me", Token: admin},
		// The listing that was N+1. It maps a page of upstream records to
		// Global IDs, so it is the one where a regression shows up first.
		{Name: "clients", Path: "/api/v1/clients?limit=50", Token: admin},
		{Name: "contacts", Path: "/api/v1/contacts?limit=50", Token: admin},
		{Name: "issues", Path: "/api/v1/issues?limit=50", Token: admin},
		// Platform-owned reads, with no upstream in the path at all.
		{Name: "notifications", Path: "/api/v1/notifications?limit=25", Token: admin},
		{Name: "servers", Path: "/api/v1/servers?limit=25", Token: devops},
		{Name: "search", Path: "/api/v1/search?q=northwind&limit=20", Token: admin},
	}

	client := &http.Client{
		Timeout: *timeout,
		Transport: &http.Transport{
			// Without this, every connection is new and the report measures
			// TCP setup rather than the platform.
			MaxIdleConnsPerHost: *concurrency,
		},
	}

	fmt.Printf("target                requests  errors      p50      p90      p99      max\n")
	fmt.Printf("--------------------------------------------------------------------------\n")

	failed := false
	for _, t := range targets {
		report, err := run(client, *base, t, *concurrency, *requests)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", t.Name, err)
			failed = true
			continue
		}
		fmt.Printf("%-20s %9d %7d %8s %8s %8s %8s\n",
			t.Name, report.Total, report.Errors,
			ms(report.P50), ms(report.P90), ms(report.P99), ms(report.Max))
		if report.Errors > 0 {
			// A load run with errors is not a slow platform, it is a broken
			// one, and the percentiles below are measured over whatever
			// happened to succeed.
			failed = true
		}
	}
	if failed {
		fmt.Fprintln(os.Stderr, "\nthe run had errors: the percentiles above describe only the requests that succeeded")
		os.Exit(1)
	}
}

type report struct {
	Total, Errors      int
	P50, P90, P99, Max time.Duration
}

func run(client *http.Client, base string, t target, concurrency, total int) (report, error) {
	if concurrency < 1 || total < 1 {
		return report{}, fmt.Errorf("concurrency and requests must be positive")
	}

	durations := make([]time.Duration, total)
	var errors int64
	var index int64

	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := atomic.AddInt64(&index, 1) - 1
				if i >= int64(total) {
					return
				}
				started := time.Now()
				status, err := once(client, base, t)
				durations[i] = time.Since(started)
				if err != nil || status >= 400 {
					atomic.AddInt64(&errors, 1)
				}
			}
		}()
	}
	wg.Wait()

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return report{
		Total:  total,
		Errors: int(errors),
		P50:    percentile(durations, 50),
		P90:    percentile(durations, 90),
		P99:    percentile(durations, 99),
		Max:    durations[len(durations)-1],
	}, nil
}

func once(client *http.Client, base string, t target) (int, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, base+t.Path, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
	if t.Token != "" {
		request.Header.Set("Authorization", "Bearer "+t.Token)
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	// The body must be drained and closed or the connection is not reused,
	// and the run then measures connection churn rather than the platform.
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode, nil
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := (p * len(sorted)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
