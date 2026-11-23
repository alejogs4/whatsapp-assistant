package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRunner struct {
	started bool
	stopped bool
}

func (f *fakeRunner) Start(ctx context.Context) error {
	f.started = true
	<-ctx.Done()
	f.stopped = true
	return nil
}

func TestLoadConfigFromEnv_FailsFastOnMissingGeminiKey(t *testing.T) {
	_, err := LoadConfigFromEnv(func(string) string { return "" })
	if !errors.Is(err, ErrMissingGeminiAPIKey) {
		t.Fatalf("expected ErrMissingGeminiAPIKey, got %v", err)
	}

	cfg, err := LoadConfigFromEnv(func(k string) string {
		if k == "GEMINI_API_KEY" {
			return "test-key"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("LoadConfigFromEnv valid: %v", err)
	}
	if cfg.SQLiteDSN == "" || cfg.WhatsAppSessionDB == "" {
		t.Fatalf("expected default DSNs, got %+v", cfg)
	}
}

func TestRunDaemon_StopsCleanlyOnContextCancel(t *testing.T) {
	sched := &fakeRunner{}
	transport := &fakeRunner{}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runDaemon(ctx, sched, transport)
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("runDaemon returned error: %v", err)
	}
	if !sched.started || !sched.stopped || !transport.started || !transport.stopped {
		t.Fatalf("expected both runners started & stopped cleanly: sched=%+v transport=%+v", sched, transport)
	}
}
