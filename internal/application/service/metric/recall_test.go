package metric

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestRecallMetric_Compute(t *testing.T) {
	tests := []struct {
		name     string
		input    *types.MetricInput
		expected float64
	}{
		{
			name: "perfect recall - all ground truth retrieved",
			input: &types.MetricInput{
				RetrievalGT:  [][]int{{1, 2, 3}},
				RetrievalIDs: []int{1, 2, 3, 4},
			},
			expected: 1.0,
		},
		{
			name: "partial recall - some ground truth retrieved",
			input: &types.MetricInput{
				RetrievalGT:  [][]int{{1, 2, 3}, {4, 5}},
				RetrievalIDs: []int{1, 4, 6},
			},
			// Hit 2 elements in the ground truth sets (a and d)
			expected: 0.41666666666666663, // (1/3 + 1/2) / 2 = 0.41666 (recall counts as long as one hit per ground truth set)

		},
		{
			name: "no recall - no ground truth retrieved",
			input: &types.MetricInput{
				RetrievalGT:  [][]int{{1, 2, 3}},
				RetrievalIDs: []int{4, 5, 6},
			},
			expected: 0.0,
		},
		{
			name: "empty retrieval list",
			input: &types.MetricInput{
				RetrievalGT:  [][]int{{1, 2, 3}},
				RetrievalIDs: []int{},
			},
			expected: 0.0,
		},
		{
			name: "multiple ground truth sets",
			input: &types.MetricInput{
				RetrievalGT:  [][]int{{1, 2}, {3, 4}, {5, 6}},
				RetrievalIDs: []int{1, 3, 7},
			},
			// Hit the first two ground truth sets (a and c)
			expected: 0.3333333333333333, // 1/3≈0.333...
		},
	}

	rm := NewRecallMetric()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.Compute(tt.input)
			if got != tt.expected {
				t.Errorf("Compute() = %v, want %v", got, tt.expected)
			}
		})
	}
}
