package usecase

import (
	"reflect"
	"testing"
)

func TestValidateAndSort(t *testing.T) {
	tests := []struct {
		name        string
		def         WorkflowDefinition
		expected    []string
		expectError bool
	}{
		{
			name: "sequential execution",
			def: WorkflowDefinition{
				Steps: []Step{
					{ID: "step2", DependsOn: []string{"step1"}},
					{ID: "step1"},
				},
			},
			expected:    []string{"step1", "step2"},
			expectError: false,
		},
		{
			name: "circular dependency",
			def: WorkflowDefinition{
				Steps: []Step{
					{ID: "step1", DependsOn: []string{"step2"}},
					{ID: "step2", DependsOn: []string{"step1"}},
				},
			},
			expectError: true,
		},
		{
			name: "empty definition",
			def: WorkflowDefinition{
				Steps: []Step{},
			},
			expectError: true,
		},
		{
			name: "missing dependency",
			def: WorkflowDefinition{
				Steps: []Step{
					{ID: "step1", DependsOn: []string{"step-nonexistent"}},
				},
			},
			expectError: true,
		},
		{
			name: "duplicate step ID",
			def: WorkflowDefinition{
				Steps: []Step{
					{ID: "step1"},
					{ID: "step1"},
				},
			},
			expectError: true,
		},
		{
			name: "empty step ID",
			def: WorkflowDefinition{
				Steps: []Step{
					{ID: ""},
				},
			},
			expectError: true,
		},
		{
			name: "complex acyclic graph",
			def: WorkflowDefinition{
				Steps: []Step{
					{ID: "D", DependsOn: []string{"B", "C"}},
					{ID: "B", DependsOn: []string{"A"}},
					{ID: "C", DependsOn: []string{"A"}},
					{ID: "A"},
				},
			},
			expected:    []string{"A", "B", "C", "D"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndSort(tt.def)
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}
			if !tt.expectError {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("expected: %v, got: %v", tt.expected, got)
				}
			}
		})
	}
}
