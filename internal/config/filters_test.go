package config_test

import (
	"strings"
	"testing"

	"github.com/hellej/pr-slack-reminder-action/internal/config"
)

func TestGetGlobalFiltersFromInput_Valid(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedFilter config.Filters
	}{
		{
			name:           "empty input",
			input:          "",
			expectedFilter: config.Filters{},
		},
		{
			name:  "authors only",
			input: `{"authors": ["alice", "bob"]}`,
			expectedFilter: config.Filters{
				Authors: []string{"alice", "bob"},
			},
		},
		{
			name:  "authors-ignore only",
			input: `{"authors-ignore": ["charlie"]}`,
			expectedFilter: config.Filters{
				AuthorsIgnore: []string{"charlie"},
			},
		},
		{
			name:  "labels only",
			input: `{"labels": ["feature", "bugfix"]}`,
			expectedFilter: config.Filters{
				Labels: []string{"feature", "bugfix"},
			},
		},
		{
			name:  "labels-ignore only",
			input: `{"labels-ignore": ["wip", "draft"]}`,
			expectedFilter: config.Filters{
				LabelsIgnore: []string{"wip", "draft"},
			},
		},
		{
			name:  "all fields",
			input: `{"authors": ["alice"], "labels": ["feature"], "labels-ignore": ["wip"]}`,
			expectedFilter: config.Filters{
				Authors:      []string{"alice"},
				Labels:       []string{"feature"},
				LabelsIgnore: []string{"wip"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setInput("test-filters", tc.input)

			filters, err := config.GetGlobalFiltersFromInput("test-filters")
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(filters.Authors) != len(tc.expectedFilter.Authors) {
				t.Errorf("Expected authors length %d, got %d", len(tc.expectedFilter.Authors), len(filters.Authors))
			}
			for i, author := range tc.expectedFilter.Authors {
				if i >= len(filters.Authors) || filters.Authors[i] != author {
					t.Errorf("Expected author[%d] '%s', got '%s'", i, author, filters.Authors[i])
				}
			}

			if len(filters.AuthorsIgnore) != len(tc.expectedFilter.AuthorsIgnore) {
				t.Errorf("Expected authors-ignore length %d, got %d", len(tc.expectedFilter.AuthorsIgnore), len(filters.AuthorsIgnore))
			}
			for i, author := range tc.expectedFilter.AuthorsIgnore {
				if i >= len(filters.AuthorsIgnore) || filters.AuthorsIgnore[i] != author {
					t.Errorf("Expected authors-ignore[%d] '%s', got '%s'", i, author, filters.AuthorsIgnore[i])
				}
			}

			if len(filters.Labels) != len(tc.expectedFilter.Labels) {
				t.Errorf("Expected labels length %d, got %d", len(tc.expectedFilter.Labels), len(filters.Labels))
			}
			for i, label := range tc.expectedFilter.Labels {
				if i >= len(filters.Labels) || filters.Labels[i] != label {
					t.Errorf("Expected label[%d] '%s', got '%s'", i, label, filters.Labels[i])
				}
			}

			if len(filters.LabelsIgnore) != len(tc.expectedFilter.LabelsIgnore) {
				t.Errorf("Expected labels-ignore length %d, got %d", len(tc.expectedFilter.LabelsIgnore), len(filters.LabelsIgnore))
			}
			for i, label := range tc.expectedFilter.LabelsIgnore {
				if i >= len(filters.LabelsIgnore) || filters.LabelsIgnore[i] != label {
					t.Errorf("Expected labels-ignore[%d] '%s', got '%s'", i, label, filters.LabelsIgnore[i])
				}
			}
		})
	}
}

func TestGetGlobalFiltersFromInput_Invalid(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedErrMsg string
	}{
		{
			name:           "invalid JSON",
			input:          `{"invalid": json}`,
			expectedErrMsg: "unable to parse filters from",
		},
		{
			name:           "unknown field",
			input:          `{"unknown": ["value"]}`,
			expectedErrMsg: "unknown field \"unknown\"",
		},
		{
			name:           "conflicting authors",
			input:          `{"authors": ["alice"], "authors-ignore": ["bob"]}`,
			expectedErrMsg: "cannot use both authors and authors-ignore filters at the same time",
		},
		{
			name:           "conflicting labels",
			input:          `{"labels": ["feature"], "labels-ignore": ["feature"]}`,
			expectedErrMsg: "labels filter cannot contain labels that are in labels-ignore filter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setInput("test-filters", tc.input)

			_, err := config.GetGlobalFiltersFromInput("test-filters")
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErrMsg, err.Error())
			}
		})
	}
}

func TestGetRepositoryFiltersFromInput_Valid(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		expectedFilters map[string]config.Filters
	}{
		{
			name:            "empty input",
			input:           "",
			expectedFilters: map[string]config.Filters{},
		},
		{
			name:  "single repository filter",
			input: `repo1: {"authors": ["alice"]}`,
			expectedFilters: map[string]config.Filters{
				"repo1": {
					Authors: []string{"alice"},
				},
			},
		},
		{
			name:  "multiple repository filters",
			input: `repo1: {"authors": ["alice"]}; repo2: {"labels-ignore": ["wip"]}`,
			expectedFilters: map[string]config.Filters{
				"repo1": {
					Authors: []string{"alice"},
				},
				"repo2": {
					LabelsIgnore: []string{"wip"},
				},
			},
		},
		{
			name:  "empty repository filter",
			input: `repo1: {}`,
			expectedFilters: map[string]config.Filters{
				"repo1": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setInput("test-repo-filters", tc.input)

			filters, err := config.GetRepositoryFiltersFromInput("test-repo-filters")
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(filters) != len(tc.expectedFilters) {
				t.Fatalf("Expected %d repository filters, got %d", len(tc.expectedFilters), len(filters))
			}

			for repo, expectedFilter := range tc.expectedFilters {
				actualFilter, exists := filters[repo]
				if !exists {
					t.Errorf("Expected repository filter for '%s' to exist", repo)
					continue
				}

				if len(actualFilter.Authors) != len(expectedFilter.Authors) {
					t.Errorf("Repository '%s': expected authors length %d, got %d", repo, len(expectedFilter.Authors), len(actualFilter.Authors))
				}
				if len(actualFilter.LabelsIgnore) != len(expectedFilter.LabelsIgnore) {
					t.Errorf("Repository '%s': expected labels-ignore length %d, got %d", repo, len(expectedFilter.LabelsIgnore), len(actualFilter.LabelsIgnore))
				}
			}
		})
	}
}

func TestGetRepositoryFiltersFromInput_Invalid(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedErrMsg string
	}{
		{
			name:           "invalid mapping format",
			input:          "invalid-format",
			expectedErrMsg: "invalid mapping format",
		},
		{
			name:           "invalid JSON in repository filter",
			input:          `repo1: {"invalid": json}`,
			expectedErrMsg: "error parsing filters for repository repo1",
		},
		{
			name:           "conflicting filters in repository",
			input:          `repo1: {"authors": ["alice"], "authors-ignore": ["bob"]}`,
			expectedErrMsg: "error parsing filters for repository repo1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setInput("test-repo-filters", tc.input)

			_, err := config.GetRepositoryFiltersFromInput("test-repo-filters")
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErrMsg, err.Error())
			}
		})
	}
}

func TestFiltersValidate(t *testing.T) {
	testCases := []struct {
		name           string
		filter         config.Filters
		shouldBeValid  bool
		expectedErrMsg string
	}{
		{
			name:          "empty filter",
			filter:        config.Filters{},
			shouldBeValid: true,
		},
		{
			name: "authors only",
			filter: config.Filters{
				Authors: []string{"alice", "bob"},
			},
			shouldBeValid: true,
		},
		{
			name: "authors-ignore only",
			filter: config.Filters{
				AuthorsIgnore: []string{"charlie"},
			},
			shouldBeValid: true,
		},
		{
			name: "labels and labels-ignore without overlap",
			filter: config.Filters{
				Labels:       []string{"feature"},
				LabelsIgnore: []string{"wip"},
			},
			shouldBeValid: true,
		},
		{
			name: "conflicting authors",
			filter: config.Filters{
				Authors:       []string{"alice"},
				AuthorsIgnore: []string{"bob"},
			},
			shouldBeValid:  false,
			expectedErrMsg: "cannot use both authors and authors-ignore filters at the same time",
		},
		{
			name: "overlapping labels",
			filter: config.Filters{
				Labels:       []string{"feature", "bug"},
				LabelsIgnore: []string{"feature", "wip"},
			},
			shouldBeValid:  false,
			expectedErrMsg: "labels filter cannot contain labels that are in labels-ignore filter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)

			var jsonStr string
			if len(tc.filter.Authors) > 0 || len(tc.filter.AuthorsIgnore) > 0 ||
				len(tc.filter.Labels) > 0 || len(tc.filter.LabelsIgnore) > 0 {
				parts := []string{}
				if len(tc.filter.Authors) > 0 {
					parts = append(parts, `"authors": ["`+strings.Join(tc.filter.Authors, `", "`)+`"]`)
				}
				if len(tc.filter.AuthorsIgnore) > 0 {
					parts = append(parts, `"authors-ignore": ["`+strings.Join(tc.filter.AuthorsIgnore, `", "`)+`"]`)
				}
				if len(tc.filter.Labels) > 0 {
					parts = append(parts, `"labels": ["`+strings.Join(tc.filter.Labels, `", "`)+`"]`)
				}
				if len(tc.filter.LabelsIgnore) > 0 {
					parts = append(parts, `"labels-ignore": ["`+strings.Join(tc.filter.LabelsIgnore, `", "`)+`"]`)
				}
				jsonStr = "{" + strings.Join(parts, ", ") + "}"
			} else {
				jsonStr = "{}"
			}

			h.setInput("test-filter", jsonStr)

			_, err := config.GetGlobalFiltersFromInput("test-filter")

			if tc.shouldBeValid {
				if err != nil {
					t.Errorf("Expected filter to be valid, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected filter to be invalid, got no error")
				} else if !strings.Contains(err.Error(), tc.expectedErrMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErrMsg, err.Error())
				}
			}
		})
	}
}
