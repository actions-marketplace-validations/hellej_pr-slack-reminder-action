package utilities

import (
	"slices"
	"testing"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name     string
		items    []int
		filter   func(int) bool
		expected []int
	}{
		{
			name:     "filter even numbers",
			items:    []int{1, 2, 3, 4, 5, 6},
			filter:   func(n int) bool { return n%2 == 0 },
			expected: []int{2, 4, 6},
		},
		{
			name:     "filter greater than 3",
			items:    []int{1, 2, 3, 4, 5},
			filter:   func(n int) bool { return n > 3 },
			expected: []int{4, 5},
		},
		{
			name:     "no items match filter",
			items:    []int{1, 3, 5},
			filter:   func(n int) bool { return n%2 == 0 },
			expected: []int{},
		},
		{
			name:     "all items match filter",
			items:    []int{2, 4, 6},
			filter:   func(n int) bool { return n%2 == 0 },
			expected: []int{2, 4, 6},
		},
		{
			name:     "empty slice",
			items:    []int{},
			filter:   func(n int) bool { return true },
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Filter(tt.items, tt.filter)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("Filter() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterWithStrings(t *testing.T) {
	items := []string{"apple", "banana", "apricot", "cherry"}
	filter := func(s string) bool { return len(s) > 5 }

	result := Filter(items, filter)
	expected := []string{"banana", "apricot", "cherry"}

	if !slices.Equal(result, expected) {
		t.Errorf("Filter() = %v, expected %v", result, expected)
	}
}
