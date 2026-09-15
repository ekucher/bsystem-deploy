// Package mockhttp provides the shared HTTP scaffolding used by the BSYSTEM
// deterministic upstream mocks.
//
// The mocks exist so that the platform can be validated end-to-end without any
// production credentials. Every secret accepted by a mock is a test-only value
// and must never be reused for a real system.
package mockhttp

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FaultMatchAll is the path value that makes an injected fault apply to every
// request handled by the server.
const FaultMatchAll = "*"

// Fault describes a single injected upstream failure.
type Fault struct {
	// Path is the request path the fault applies to, or FaultMatchAll.
	Path string `json:"path"`
	// Status is the HTTP status code returned instead of the normal response.
	// It may be zero when only a delay is injected.
	Status int `json:"status,omitempty"`
	// Body is an optional raw response body. When empty a generic JSON error
	// body is produced.
	Body string `json:"body,omitempty"`
	// DelayMS delays the response, which lets tests exercise adapter timeouts.
	DelayMS int `json:"delay_ms,omitempty"`
	// Remaining is how many requests the fault applies to. Zero means one.
	Remaining int `json:"remaining,omitempty"`
}

// Faults is a concurrency-safe queue of injected faults keyed by request path.
type Faults struct {
	mu     sync.Mutex
	queues map[string][]Fault
}

// NewFaults returns an empty fault controller.
func NewFaults() *Faults {
	return &Faults{queues: map[string][]Fault{}}
}

// Add queues a fault. An empty path applies the fault to every request.
func (f *Faults) Add(fault Fault) error {
	fault.Path = strings.TrimSpace(fault.Path)
	if fault.Path == "" {
		fault.Path = FaultMatchAll
	}
	if fault.Status == 0 && fault.DelayMS == 0 {
		return errors.New("fault requires status or delay_ms")
	}
	if fault.Status != 0 && (fault.Status < 100 || fault.Status > 599) {
		return errors.New("fault status must be a valid HTTP status code")
	}
	if fault.DelayMS < 0 {
		return errors.New("delay_ms must not be negative")
	}
	if fault.Remaining <= 0 {
		fault.Remaining = 1
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queues[fault.Path] = append(f.queues[fault.Path], fault)
	return nil
}

// Reset removes every queued fault.
func (f *Faults) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queues = map[string][]Fault{}
}

// Pending returns the queued faults, sorted by path, for introspection.
func (f *Faults) Pending() []Fault {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := []Fault{}
	for _, queue := range f.queues {
		result = append(result, queue...)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

// Take consumes the next fault applicable to path, preferring an exact path
// match over a catch-all entry.
func (f *Faults) Take(path string) (Fault, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, key := range []string{path, FaultMatchAll} {
		queue := f.queues[key]
		if len(queue) == 0 {
			continue
		}
		fault := queue[0]
		fault.Remaining--
		if fault.Remaining <= 0 {
			f.queues[key] = queue[1:]
		} else {
			queue[0] = fault
			f.queues[key] = queue
		}
		return fault, true
	}
	return Fault{}, false
}

// WriteJSON writes v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a small JSON error body with the given status.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{"error": message, "status": status})
}

// Server wires the shared behaviour every BSYSTEM mock upstream needs:
// deterministic routing, a health endpoint, fault injection and
// credential-safe request logging.
type Server struct {
	Name   string
	Faults *Faults

	mux *http.ServeMux
}

// NewServer returns a mock server named name.
func NewServer(name string) *Server {
	s := &Server{Name: name, Faults: NewFaults(), mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": s.Name})
	})
	s.mux.HandleFunc("POST /__mock/faults", s.addFault)
	s.mux.HandleFunc("GET /__mock/faults", s.listFaults)
	s.mux.HandleFunc("DELETE /__mock/faults", s.resetFaults)
	return s
}

// Handle registers a handler for a net/http routing pattern.
func (s *Server) Handle(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

// Handler returns the fully wrapped handler.
func (s *Server) Handler() http.Handler {
	return s.logging(s.injectFaults(s.mux))
}

func (s *Server) addFault(w http.ResponseWriter, r *http.Request) {
	var fault Fault
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&fault); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.Faults.Add(fault); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, fault)
}

func (s *Server) listFaults(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, s.Faults.Pending())
}

func (s *Server) resetFaults(w http.ResponseWriter, _ *http.Request) {
	s.Faults.Reset()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) injectFaults(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/__mock/") || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		fault, ok := s.Faults.Take(r.URL.Path)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if fault.DelayMS > 0 {
			select {
			case <-time.After(time.Duration(fault.DelayMS) * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
		if fault.Status == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if fault.Status == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", "1")
		}
		if fault.Body != "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(fault.Status)
			_, _ = w.Write([]byte(fault.Body))
			return
		}
		WriteError(w, fault.Status, "injected mock fault")
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

// logging records method, path and status only. Request headers and query
// values are never logged because they can carry test credentials.
func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s %s %d %s", s.Name, r.Method, r.URL.Path, recorder.status, time.Since(start).Round(time.Millisecond))
	})
}

// ListenAndServe starts the mock on the address from addrEnv, defaulting to
// defaultAddr, with bounded server timeouts.
func (s *Server) ListenAndServe(addrEnv, defaultAddr string) error {
	addr := strings.TrimSpace(os.Getenv(addrEnv))
	if addr == "" {
		addr = defaultAddr
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// Write timeout must exceed the largest delay a timeout scenario
		// injects, otherwise the mock cuts the response before the adapter
		// gets to demonstrate its own timeout handling.
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("%s listening on %s", s.Name, addr)
	return server.ListenAndServe()
}

// Page resolves an offset/limit window over total items, clamping both to safe
// bounds. It returns the start and end indices of the window.
func Page(total, offset, limit, defaultLimit, maxLimit int) (int, int) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return offset, end
}

// QueryInt reads a non-negative integer query parameter, returning fallback
// when the parameter is missing or malformed.
func QueryInt(r *http.Request, name string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

// Secret reads a test-only credential from the environment, falling back to a
// documented default so the mocks run with no configuration at all.
func Secret(env, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(env)); value != "" {
		return value
	}
	return fallback
}

// RequireHeader enforces an exact header value and writes a 401 when it does
// not match. The expected and supplied values are never logged.
func RequireHeader(w http.ResponseWriter, r *http.Request, header, expected string) bool {
	if r.Header.Get(header) == expected {
		return true
	}
	WriteError(w, http.StatusUnauthorized, "invalid or missing credential")
	return false
}

// RequireBearer enforces an exact bearer token and writes a 401 when it does
// not match.
func RequireBearer(w http.ResponseWriter, r *http.Request, expected string) bool {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") && strings.TrimPrefix(header, "Bearer ") == expected {
		return true
	}
	WriteError(w, http.StatusUnauthorized, "invalid or missing credential")
	return false
}

// MaybeHealthCheck implements the container healthcheck for the mock images.
//
// The images ship as scratch containers with no shell and no curl, so the mock
// binary probes itself when it is invoked as `<binary> -healthcheck`. It exits
// the process rather than returning when the flag is present.
func MaybeHealthCheck(defaultAddr string) {
	if len(os.Args) < 2 || os.Args[1] != "-healthcheck" {
		return
	}
	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		log.Printf("healthcheck failed: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("healthcheck failed: HTTP %d", resp.StatusCode)
		os.Exit(1)
	}
	os.Exit(0)
}
