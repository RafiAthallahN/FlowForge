package usecase

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type mockStepRunner struct {
	behavior func(ctx context.Context, step Step) (string, error)
}

func (m *mockStepRunner) Run(ctx context.Context, step Step) (string, error) {
	if m.behavior != nil {
		return m.behavior(ctx, step)
	}
	return "ok", nil
}

func TestEngineSuccess(t *testing.T) {
	eng := NewEngine(&mockStepRunner{})
	def := WorkflowDefinition{
		Steps: []Step{{ID: "s1"}},
	}
	res, err := eng.Execute(context.Background(), def, time.Second)
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected success, got %v", res.Status)
	}
}

func TestEngineTimeout(t *testing.T) {
	runner := &mockStepRunner{
		behavior: func(ctx context.Context, step Step) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return "ok", nil
			}
		},
	}
	eng := NewEngine(runner)
	def := WorkflowDefinition{
		Steps: []Step{{ID: "s1"}},
	}
	res, err := eng.Execute(context.Background(), def, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	if res.Status != StatusFailed {
		t.Errorf("expected failure, got %v", res.Status)
	}
}

func TestEngineRetryPolicy(t *testing.T) {
	var attempts int32
	runner := &mockStepRunner{
		behavior: func(ctx context.Context, step Step) (string, error) {
			count := atomic.AddInt32(&attempts, 1)
			if count < 3 {
				return "", errors.New("temporary error")
			}
			return "success output", nil
		},
	}
	eng := NewEngine(runner)
	def := WorkflowDefinition{
		Steps: []Step{
			{
				ID: "s1",
				RetryPolicy: RetryPolicy{
					MaxRetries:        3,
					InitialIntervalMS: 1,
					BackoffFactor:     2.0,
				},
			},
		},
	}
	res, err := eng.Execute(context.Background(), def, time.Second)
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected success, got %v", res.Status)
	}

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts != 3 {
		t.Errorf("expected exactly 3 attempts (initial + 2 retries), got %d", finalAttempts)
	}

	log, ok := res.StepLogs["s1"]
	if !ok {
		t.Fatal("step s1 log not found")
	}
	if log.RetryCount != 2 {
		t.Errorf("expected log retry count to be 2, got %d", log.RetryCount)
	}
	if log.Status != StatusSuccess {
		t.Errorf("expected step status Success, got %v", log.Status)
	}
}

func TestEngineDependencyFailure(t *testing.T) {
	runner := &mockStepRunner{
		behavior: func(ctx context.Context, step Step) (string, error) {
			if step.ID == "s1" {
				return "", errors.New("s1 failed")
			}
			return "s2 ok", nil
		},
	}
	eng := NewEngine(runner)
	def := WorkflowDefinition{
		Steps: []Step{
			{ID: "s1"},
			{ID: "s2", DependsOn: []string{"s1"}},
		},
	}
	res, err := eng.Execute(context.Background(), def, time.Second)
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	if res.Status != StatusFailed {
		t.Errorf("expected workflow status Failed, got %v", res.Status)
	}

	log1, ok := res.StepLogs["s1"]
	if !ok {
		t.Fatal("s1 log not found")
	}
	if log1.Status != StatusFailed {
		t.Errorf("expected s1 status Failed, got %v", log1.Status)
	}

	log2, ok := res.StepLogs["s2"]
	if !ok {
		t.Fatal("s2 log not found")
	}
	if log2.Status != StatusFailed {
		t.Errorf("expected s2 status Failed, got %v", log2.Status)
	}
}

func TestEngineDefaultStepRunner(t *testing.T) {
	eng := NewEngine(nil) // uses DefaultStepRunner
	def := WorkflowDefinition{
		Steps: []Step{
			{
				ID:     "s1",
				Type:   StepDelay,
				Config: 5 * time.Millisecond,
			},
			{
				ID:   "s2",
				Type: StepHTTP, // default case
			},
		},
	}
	res, err := eng.Execute(context.Background(), def, time.Second)
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected success, got %v", res.Status)
	}
}
