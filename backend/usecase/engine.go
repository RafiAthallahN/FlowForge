package usecase

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

type StepStatus string

const (
	StatusPending StepStatus = "Pending"
	StatusRunning StepStatus = "Running"
	StatusSuccess StepStatus = "Success"
	StatusFailed  StepStatus = "Failed"
)

type StepLog struct {
	StepID     string
	Status     StepStatus
	RetryCount int
	DurationMS int64
	LogOutput  string
	Err        error
}

type RunResult struct {
	Status       StepStatus
	StepLogs     map[string]*StepLog
	ErrorMessage string
	DurationMS   int64
}

type StepRunner interface {
	Run(ctx context.Context, step Step) (string, error)
}

type DefaultStepRunner struct{}

func (r *DefaultStepRunner) Run(ctx context.Context, step Step) (string, error) {
	switch step.Type {
	case StepDelay:
		duration := 10 * time.Millisecond
		if val, ok := step.Config.(time.Duration); ok {
			duration = val
		}
		timer := time.NewTimer(duration)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timer.C:
			return fmt.Sprintf("Delayed for %v", duration), nil
		}
	case StepType("fail"):
		return "", fmt.Errorf("intentional step failure for testing")
	default:
		return fmt.Sprintf("Executed step %s of type %s", step.ID, step.Type), nil
	}
}

type Engine struct {
	runner             StepRunner
	OnStepStatusChange func(stepID string, status StepStatus, logLine string, durationMS int64)
	sem                chan struct{}
}

func NewEngine(runner StepRunner) *Engine {
	if runner == nil {
		runner = &DefaultStepRunner{}
	}
	return &Engine{
		runner: runner,
		sem:    make(chan struct{}, 20), // Default limit of 20 concurrent executing steps
	}
}

type stepState struct {
	done chan struct{}
	log  *StepLog
	mu   sync.Mutex
}

func (e *Engine) Execute(ctx context.Context, def WorkflowDefinition, timeout time.Duration) (*RunResult, error) {
	sorted, err := ValidateAndSort(def)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()

	states := make(map[string]*stepState)
	for _, step := range def.Steps {
		states[step.ID] = &stepState{
			done: make(chan struct{}),
			log: &StepLog{
				StepID: step.ID,
				Status: StatusPending,
			},
		}
	}

	for _, step := range def.Steps {
		go func(s Step) {
			state := states[s.ID]
			defer close(state.done)

			for _, depID := range s.DependsOn {
				depState := states[depID]
				select {
				case <-runCtx.Done():
					state.mu.Lock()
					state.log.Status = StatusFailed
					state.log.Err = runCtx.Err()
					state.log.LogOutput = "Workflow canceled or timed out."
					state.mu.Unlock()
					if e.OnStepStatusChange != nil {
						e.OnStepStatusChange(s.ID, StatusFailed, "Workflow canceled or timed out.", 0)
					}
					return
				case <-depState.done:
					depState.mu.Lock()
					depErr := depState.log.Err
					depState.mu.Unlock()

					if depErr != nil {
						state.mu.Lock()
						state.log.Status = StatusFailed
						state.log.Err = fmt.Errorf("dependency %s failed: %w", depID, depErr)
						state.log.LogOutput = fmt.Sprintf("Skipped because dependency %s failed.", depID)
						state.mu.Unlock()
						if e.OnStepStatusChange != nil {
							e.OnStepStatusChange(s.ID, StatusFailed, fmt.Sprintf("Skipped because dependency %s failed.", depID), 0)
						}
						return
					}
				}
			}

			// Acquire concurrency token before executing
			select {
			case <-runCtx.Done():
				state.mu.Lock()
				state.log.Status = StatusFailed
				state.log.Err = runCtx.Err()
				state.log.LogOutput = "Workflow canceled or timed out."
				state.mu.Unlock()
				if e.OnStepStatusChange != nil {
					e.OnStepStatusChange(s.ID, StatusFailed, "Workflow canceled or timed out.", 0)
				}
				return
			case e.sem <- struct{}{}:
			}

			e.executeWithRetry(runCtx, s, state)

			// Release concurrency token
			<-e.sem
		}(step)
	}

	for _, id := range sorted {
		<-states[id].done
	}

	totalDuration := time.Since(startTime).Milliseconds()

	logs := make(map[string]*StepLog)
	workflowFailed := false
	var firstErr string

	for id, state := range states {
		state.mu.Lock()
		logs[id] = state.log
		if state.log.Err != nil {
			workflowFailed = true
			if firstErr == "" {
				firstErr = state.log.Err.Error()
			}
		}
		state.mu.Unlock()
	}

	result := &RunResult{
		Status:     StatusSuccess,
		StepLogs:   logs,
		DurationMS: totalDuration,
	}

	if workflowFailed {
		result.Status = StatusFailed
		result.ErrorMessage = firstErr
	}

	return result, nil
}

func (e *Engine) executeWithRetry(ctx context.Context, step Step, state *stepState) {
	state.mu.Lock()
	state.log.Status = StatusRunning
	state.mu.Unlock()
	if e.OnStepStatusChange != nil {
		e.OnStepStatusChange(step.ID, StatusRunning, "", 0)
	}

	maxRetries := step.RetryPolicy.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	interval := time.Duration(step.RetryPolicy.InitialIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}

	factor := step.RetryPolicy.BackoffFactor
	if factor <= 0 || math.IsNaN(factor) {
		factor = 2.0
	}

	var lastErr error
	var output string

	for attempt := 0; attempt <= maxRetries; attempt++ {
		state.mu.Lock()
		state.log.RetryCount = attempt
		state.mu.Unlock()

		if err := ctx.Err(); err != nil {
			state.mu.Lock()
			state.log.Status = StatusFailed
			state.log.Err = err
			state.log.LogOutput = "Cancelled before execution."
			state.mu.Unlock()
			if e.OnStepStatusChange != nil {
				e.OnStepStatusChange(step.ID, StatusFailed, "Cancelled before execution.", 0)
			}
			return
		}

		stepStart := time.Now()
		output, lastErr = e.runner.Run(ctx, step)
		duration := time.Since(stepStart).Milliseconds()

		state.mu.Lock()
		state.log.DurationMS += duration
		state.log.LogOutput += fmt.Sprintf("[Attempt %d] Duration: %dms, Output: %s\n", attempt, duration, output)
		if lastErr != nil {
			state.log.LogOutput += fmt.Sprintf("[Attempt %d] Error: %s\n", attempt, lastErr.Error())
		}
		state.mu.Unlock()

		if lastErr == nil {
			state.mu.Lock()
			state.log.Status = StatusSuccess
			state.mu.Unlock()
			if e.OnStepStatusChange != nil {
				e.OnStepStatusChange(step.ID, StatusSuccess, output, duration)
			}
			return
		}

		if attempt < maxRetries {
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				state.mu.Lock()
				state.log.Status = StatusFailed
				state.log.Err = ctx.Err()
				state.mu.Unlock()
				if e.OnStepStatusChange != nil {
					e.OnStepStatusChange(step.ID, StatusFailed, "Context done during retry backoff.", 0)
				}
				return
			case <-timer.C:
				nextInterval := float64(interval) * factor
				const maxDuration = float64(math.MaxInt64)
				if nextInterval >= maxDuration || math.IsNaN(nextInterval) {
					interval = math.MaxInt64
				} else {
					interval = time.Duration(nextInterval)
				}
			}
		}
	}

	state.mu.Lock()
	state.log.Status = StatusFailed
	state.log.Err = fmt.Errorf("step failed after %d retries: %w", maxRetries, lastErr)
	state.mu.Unlock()
	if e.OnStepStatusChange != nil {
		e.OnStepStatusChange(step.ID, StatusFailed, fmt.Sprintf("Step failed after %d retries. Last error: %v", maxRetries, lastErr), 0)
	}
}
