package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ErrorClass string

const (
	ErrorClassOK               ErrorClass = "ok"
	ErrorClassDialError        ErrorClass = "dial_error"
	ErrorClassTimeout          ErrorClass = "timeout"
	ErrorClassRemoteDisconnect ErrorClass = "remote_disconnect"
	ErrorClassLocalError       ErrorClass = "local_error"
)

type TCPSessionRecord struct {
	RunID      string `json:"run_id"`
	CaseID     string `json:"case_id"`
	TargetID   string `json:"target_id"`
	TargetRaw  string `json:"target_raw"`
	Endpoint   string `json:"endpoint"`
	StartTime  string `json:"start_time"`
	DialStart  string `json:"dial_start"`
	DialEnd    string `json:"dial_end"`
	EndTime    string `json:"end_time"`
	LatencyMS  int64  `json:"latency_ms"`
	OK         bool   `json:"ok"`
	ErrorClass string `json:"error_class"`
	Error      string `json:"error,omitempty"`
}

type TCPProbeParams struct {
	RunID       string
	CaseID      string
	Timeout     time.Duration
	Concurrency int
	Retries     int
	RecordDir   string
}

type TCPProbeTarget struct {
	ID       string
	Raw      string
	Endpoint string
}

func RunTCPProbe(ctx context.Context, p TCPProbeParams, targets []TCPProbeTarget, onResult func(TCPSessionRecord)) error {
	if p.RunID == "" {
		p.RunID = NewRunID()
	}
	if p.CaseID == "" {
		p.CaseID = "probe.tcp"
	}
	if p.Timeout <= 0 {
		p.Timeout = 3 * time.Second
	}
	if p.Concurrency <= 0 {
		p.Concurrency = 1
	}
	if p.Retries < 0 {
		p.Retries = 0
	}
	if onResult == nil {
		onResult = func(TCPSessionRecord) {}
	}

	sem := make(chan struct{}, p.Concurrency)
	var wg sync.WaitGroup
	for _, t := range targets {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				class := ErrorClassLocalError
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					class = ErrorClassTimeout
				}
				onResult(errorRecord(p, t, class, ctx.Err()))
				return
			}
			defer func() { <-sem }()

			r := probeOne(ctx, p, t)
			if p.RecordDir != "" {
				_ = WriteTCPRecord(p.RecordDir, p.RunID, p.CaseID, t.ID, r)
			}
			onResult(r)
		}()
	}
	wg.Wait()
	return nil
}

func probeOne(ctx context.Context, p TCPProbeParams, t TCPProbeTarget) TCPSessionRecord {
	start := time.Now().UTC()
	r := TCPSessionRecord{
		RunID:      p.RunID,
		CaseID:     p.CaseID,
		TargetID:   t.ID,
		TargetRaw:  t.Raw,
		Endpoint:   t.Endpoint,
		StartTime:  start.Format(time.RFC3339Nano),
		ErrorClass: string(ErrorClassOK),
	}

	var lastErr error
	var lastClass ErrorClass
	for attempt := 0; attempt <= p.Retries; attempt++ {
		sessCtx, cancel := context.WithTimeout(ctx, p.Timeout)

		r.DialStart = time.Now().UTC().Format(time.RFC3339Nano)
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(sessCtx, "tcp", t.Endpoint)
		r.DialEnd = time.Now().UTC().Format(time.RFC3339Nano)

		if err == nil {
			_ = conn.Close()
			cancel()
			r.EndTime = time.Now().UTC().Format(time.RFC3339Nano)
			r.LatencyMS = time.Since(start).Milliseconds()
			r.OK = true
			r.ErrorClass = string(ErrorClassOK)
			r.Error = ""
			return r
		}

		lastErr = err
		lastClass = classifyDialError(sessCtx, err)
		cancel()
		if lastClass != ErrorClassDialError && lastClass != ErrorClassTimeout {
			break
		}
	}

	r.EndTime = time.Now().UTC().Format(time.RFC3339Nano)
	r.LatencyMS = time.Since(start).Milliseconds()
	r.OK = false
	if lastErr != nil {
		r.Error = lastErr.Error()
	}
	r.ErrorClass = string(lastClass)
	return r
}

func classifyDialError(ctx context.Context, err error) ErrorClass {
	if err == nil {
		return ErrorClassOK
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return ErrorClassLocalError
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ErrorClassTimeout
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return ErrorClassTimeout
	}
	return ErrorClassDialError
}

func errorRecord(p TCPProbeParams, t TCPProbeTarget, class ErrorClass, err error) TCPSessionRecord {
	start := time.Now().UTC()
	r := TCPSessionRecord{
		RunID:      p.RunID,
		CaseID:     p.CaseID,
		TargetID:   t.ID,
		TargetRaw:  t.Raw,
		Endpoint:   t.Endpoint,
		StartTime:  start.Format(time.RFC3339Nano),
		DialStart:  start.Format(time.RFC3339Nano),
		DialEnd:    start.Format(time.RFC3339Nano),
		EndTime:    start.Format(time.RFC3339Nano),
		LatencyMS:  0,
		OK:         false,
		ErrorClass: string(class),
	}
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

func NewRunID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return time.Now().UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(b)
}

func WriteTCPRecord(dir, runID, caseID, targetID string, rec TCPSessionRecord) error {
	caseID = SafePathComponent(caseID)
	targetID = SafePathComponent(targetID)
	path := filepath.Join(dir, runID, caseID)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	return writeJSONFile(filepath.Join(path, targetID+".json"), rec)
}

func SafePathComponent(s string) string {
	if s == "" {
		return "_"
	}
	r := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		" ", "_",
		"\t", "_",
		"\n", "_",
	)
	return r.Replace(s)
}
