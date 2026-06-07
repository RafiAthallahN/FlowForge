package usecase

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type mockTransport struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTrip(req)
}

func TestErrorAnalyzerFallback(t *testing.T) {
	os.Setenv("OPENROUTER_API_KEY", "")
	
	analyzer := NewOpenRouterAnalyzer()
	analysis, err := analyzer.AnalyzeFailure(context.Background(), "fetch-step", "delay", "context deadline exceeded", `{"duration": 5000}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(analysis.Reason, "fetch-step") {
		t.Errorf("expected reason to contain step ID, got: %s", analysis.Reason)
	}
	if !strings.Contains(analysis.SuggestedFix, "parameters") {
		t.Errorf("expected suggested fix to contain help details, got: %s", analysis.SuggestedFix)
	}
}

func TestErrorAnalyzerAPI(t *testing.T) {
	os.Setenv("OPENROUTER_API_KEY", "test-api-key")
	defer os.Unsetenv("OPENROUTER_API_KEY")

	analyzer := NewOpenRouterAnalyzer()

	// Intercept HTTP request using mock transport
	analyzer.client.Transport = &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") != "Bearer test-api-key" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       http.NoBody,
				}, nil
			}

			w := httptest.NewRecorder()
			response := map[string]any{
				"choices": []map[string]any{
					{
						"message": map[string]string{
							"content": `{"reason": "Database connection timed out", "suggested_fix": "Increase context execution limit or verify db CPU load"}`,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return w.Result(), nil
		},
	}

	analysis, err := analyzer.AnalyzeFailure(context.Background(), "db-query", "sql", "connection timeout", `{"query": "SELECT 1"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analysis.Reason != "Database connection timed out" {
		t.Errorf("expected specific reason, got: %s", analysis.Reason)
	}
	if analysis.SuggestedFix != "Increase context execution limit or verify db CPU load" {
		t.Errorf("expected specific suggested fix, got: %s", analysis.SuggestedFix)
	}
}
