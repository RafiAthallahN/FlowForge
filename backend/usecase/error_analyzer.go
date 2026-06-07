package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// FailureAnalysis holds the diagnostics returned by the LLM.
type FailureAnalysis struct {
	Reason       string `json:"reason"`
	SuggestedFix string `json:"suggested_fix"`
}

// ErrorAnalyzer defines the service to diagnose terminal step failures.
type ErrorAnalyzer interface {
	AnalyzeFailure(ctx context.Context, stepID string, stepType string, errMsg string, payload string) (*FailureAnalysis, error)
}

// OpenRouterAnalyzer is an implementation of ErrorAnalyzer using OpenRouter's API.
type OpenRouterAnalyzer struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenRouterAnalyzer creates a new OpenRouterAnalyzer instance.
func NewOpenRouterAnalyzer() *OpenRouterAnalyzer {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "google/gemini-2.5-flash"
	}
	return &OpenRouterAnalyzer{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// AnalyzeFailure connects to OpenRouter to analyze the failure or uses a local fallback diagnostics response.
func (a *OpenRouterAnalyzer) AnalyzeFailure(ctx context.Context, stepID string, stepType string, errMsg string, payload string) (*FailureAnalysis, error) {
	// Fallback to local mock analysis if no API key is specified
	if a.apiKey == "" {
		return &FailureAnalysis{
			Reason:       fmt.Sprintf("Local Diagnostic: Step '%s' of type '%s' failed. Error: %s", stepID, stepType, errMsg),
			SuggestedFix: "Ensure all parameters in the execution payload are valid. Check that downstream database or API servers are responsive. Verify execution timeouts.",
		}, nil
	}

	prompt := fmt.Sprintf(`You are an expert site reliability engineer diagnosing a failed step in a data workflow.
Step ID: %s
Step Type: %s
Error Message: %s
Execution Payload / Config: %s

Analyze the error and provide the root cause and a suggested fix in a valid JSON object matching this schema:
{
  "reason": "Clear explanation of why this specific step failed",
  "suggested_fix": "Concrete, actionable step-by-step instructions to resolve the error"
}`, stepID, stepType, errMsg, payload)

	requestPayload := map[string]any{
		"model": a.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("HTTP-Referer", "https://flowforger.com")
	req.Header.Set("X-Title", "FlowForge")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("openrouter API status %d: %v", resp.StatusCode, errResp)
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode openrouter response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("openrouter returned no completions")
	}

	var analysis FailureAnalysis
	rawContent := chatResp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(rawContent), &analysis); err != nil {
		// Fallback parsing: if the model returned raw text instead of structured JSON
		return &FailureAnalysis{
			Reason:       rawContent,
			SuggestedFix: "Verify logs for details.",
		}, nil
	}

	return &analysis, nil
}
