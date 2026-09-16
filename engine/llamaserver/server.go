// Package llamaserver runs llama.cpp's llama-server as a child process: it
// starts the server on demand, waits for it to load, hands out authenticated
// loopback endpoints, stops it after an idle period, restarts it after a crash,
// and guarantees it never outlives Spotfile.
package llamaserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"sync"
	"time"
)

// ErrClosed is returned by Acquire after Close.
var ErrClosed = errors.New("llamaserver: server closed")

const (
	defaultStartTimeout = 2 * time.Minute // large models can take a while to load
	stopGrace           = 5 * time.Second // time to exit after a graceful stop before a forced kill
	healthPollInterval  = 100 * time.Millisecond
	healthRequestLimit  = 2 * time.Second
	logTailLines        = 20
	apiKeyBytes         = 32
)

// Config describes one llama-server instance, such as the embedding server or
// the chat server.
type Config struct {
	// Binary is the llama-server executable. Empty means Locate().
	Binary string
	// Args are model and role flags (for example -m, --embeddings). Host, port
	// and API key are always chosen by the Server and must not be included.
	Args []string
	// IdleTimeout stops the process after this long with no active leases.
	// Zero keeps it running until Close.
	IdleTimeout time.Duration
	// StartTimeout bounds the wait for /health after launch. Zero means 2 minutes.
	StartTimeout time.Duration
}

// Server manages the lifecycle of one llama-server process. It is safe for
// concurrent use.
type Server struct {
	cfg Config

	mu     sync.Mutex
	proc   *process
	idle   *time.Timer
	closed bool
}

// Lease is an authenticated endpoint on a running server. The server will not
// be stopped for idleness while any lease is held; call Release when done.
type Lease struct {
	BaseURL string // e.g. http://127.0.0.1:53211
	APIKey  string // send as "Authorization: Bearer <APIKey>"

	once    sync.Once
	release func()
}

// Release returns the lease. Calling it more than once is harmless.
func (l *Lease) Release() { l.once.Do(l.release) }

type process struct {
	cmd     *exec.Cmd
	pl      platform
	baseURL string
	apiKey  string
	logs    *tailBuffer
	leases  int // guarded by Server.mu

	ready chan struct{} // closed once healthy or failed; err is final by then
	err   error
	done  chan struct{} // closed when the process has exited

	stopOnce sync.Once
}

// New returns a Server. No process is started until the first Acquire.
func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// Acquire returns a lease on a healthy server, starting (or restarting) the
// process if needed. Concurrent callers share a single process.
func (s *Server) Acquire(ctx context.Context) (*Lease, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	p := s.proc
	if p == nil || isClosed(p.done) {
		p = s.launch()
		s.proc = p
	}
	p.leases++
	s.mu.Unlock()

	select {
	case <-p.ready:
	case <-ctx.Done():
		s.release(p)
		return nil, ctx.Err()
	}
	if p.err != nil {
		s.release(p)
		return nil, p.err
	}
	return &Lease{BaseURL: p.baseURL, APIKey: p.apiKey, release: func() { s.release(p) }}, nil
}

// Running reports whether a healthy server process is currently up.
func (s *Server) Running() bool {
	s.mu.Lock()
	p := s.proc
	s.mu.Unlock()
	return p != nil && isClosed(p.ready) && p.err == nil && !isClosed(p.done)
}

// Close stops the process and makes further Acquire calls fail with ErrClosed.
func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	p := s.proc
	s.proc = nil
	s.mu.Unlock()

	if p != nil {
		p.stop()
	}
	return nil
}

// launch starts a process and returns immediately; startup success or failure
// is reported through p.ready and p.err. Must be called with s.mu held.
func (s *Server) launch() *process {
	p := &process{
		ready: make(chan struct{}),
		done:  make(chan struct{}),
		logs:  newTailBuffer(logTailLines),
	}
	fail := func(err error) *process {
		p.err = err
		close(p.ready)
		close(p.done)
		return p
	}

	bin := s.cfg.Binary
	if bin == "" {
		var err error
		if bin, err = Locate(); err != nil {
			return fail(err)
		}
	}
	if _, err := os.Stat(bin); err != nil {
		return fail(fmt.Errorf("llama-server binary: %w", err))
	}
	port, err := freePort()
	if err != nil {
		return fail(fmt.Errorf("reserve port: %w", err))
	}
	key, err := newAPIKey()
	if err != nil {
		return fail(fmt.Errorf("generate api key: %w", err))
	}
	p.baseURL = "http://127.0.0.1:" + strconv.Itoa(port)
	p.apiKey = key

	args := append(slices.Clone(s.cfg.Args),
		"--host", "127.0.0.1", "--port", strconv.Itoa(port), "--api-key", key)
	cmd, pl, err := startCommand(bin, args, p.logs)
	if err != nil {
		return fail(fmt.Errorf("start llama-server: %w", err))
	}
	p.cmd, p.pl = cmd, pl

	go func() {
		_ = cmd.Wait()
		pl.release()
		close(p.done)
		s.forget(p)
	}()
	go s.awaitHealthy(p)
	return p
}

// awaitHealthy polls /health until the server is ready, exits, or the start
// timeout passes. A timed-out process is stopped.
func (s *Server) awaitHealthy(p *process) {
	timeout := s.cfg.StartTimeout
	if timeout <= 0 {
		timeout = defaultStartTimeout
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(healthPollInterval)
	defer tick.Stop()
	client := &http.Client{Timeout: healthRequestLimit}

	for {
		if healthy(client, p) {
			close(p.ready)
			return
		}
		select {
		case <-p.done:
			p.err = fmt.Errorf("llama-server exited during startup (%s):\n%s", p.cmd.ProcessState, p.logs)
			close(p.ready)
			return
		case <-deadline.C:
			p.err = fmt.Errorf("llama-server not ready after %s:\n%s", timeout, p.logs)
			close(p.ready)
			s.forget(p)
			p.stop()
			return
		case <-tick.C:
		}
	}
}

func healthy(client *http.Client, p *process) bool {
	req, err := http.NewRequest(http.MethodGet, p.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// release drops one lease and arms the idle timer when none remain.
func (s *Server) release(p *process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.leases--
	if s.closed || s.proc != p || p.leases > 0 || s.cfg.IdleTimeout <= 0 {
		return
	}
	if s.idle != nil {
		s.idle.Stop()
	}
	s.idle = time.AfterFunc(s.cfg.IdleTimeout, func() { s.stopIfIdle(p) })
}

func (s *Server) stopIfIdle(p *process) {
	s.mu.Lock()
	if s.proc != p || p.leases > 0 {
		s.mu.Unlock()
		return
	}
	s.proc = nil
	s.idle = nil
	s.mu.Unlock()
	p.stop()
}

// forget clears p as the current process so the next Acquire starts a new one.
func (s *Server) forget(p *process) {
	s.mu.Lock()
	if s.proc == p {
		s.proc = nil
	}
	s.mu.Unlock()
}

// stop asks the process to exit, then forces it after stopGrace.
func (p *process) stop() {
	p.stopOnce.Do(func() {
		if p.cmd == nil {
			return
		}
		p.pl.terminate(p.cmd)
		select {
		case <-p.done:
		case <-time.After(stopGrace):
			p.pl.kill(p.cmd)
			<-p.done
		}
	})
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func newAPIKey() (string, error) {
	b := make([]byte, apiKeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
