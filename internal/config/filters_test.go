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
			name:  "ignored-authors only",
			input: `{"ignored-authors": ["charlie"]}`,
			expectedFilter: config.Filters{
				IgnoredAuthors: []string{"charlie"},
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
			name:  "ignored-labels only",
			input: `{"ignored-labels": ["wip", "draft"]}`,
			expectedFilter: config.Filters{
				IgnoredLabels: []string{"wip", "draft"},
			},
		},
		{
			name:  "ignored-terms only",
			input: `{"ignored-terms": ["Release v1.0", "Automated Update"]}`,
			expectedFilter: config.Filters{
				IgnoredTerms: []string{"Release v1.0", "Automated Update"},
			},
		},
		{
			name:  "all fields",
			input: `{"authors": ["alice"], "labels": ["feature"], "ignored-labels": ["wip"]}`,
			expectedFilter: config.Filters{
				Authors:       []string{"alice"},
				Labels:        []string{"feature"},
				IgnoredLabels: []string{"wip"},
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

			if len(filters.IgnoredAuthors) != len(tc.expectedFilter.IgnoredAuthors) {
				t.Errorf("Expected ignored-authors length %d, got %d", len(tc.expectedFilter.IgnoredAuthors), len(filters.IgnoredAuthors))
			}
			for i, author := range tc.expectedFilter.IgnoredAuthors {
				if i >= len(filters.IgnoredAuthors) || filters.IgnoredAuthors[i] != author {
					t.Errorf("Expected ignored-authors[%d] '%s', got '%s'", i, author, filters.IgnoredAuthors[i])
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

			if len(filters.IgnoredLabels) != len(tc.expectedFilter.IgnoredLabels) {
				t.Errorf("Expected ignored-labels length %d, got %d", len(tc.expectedFilter.IgnoredLabels), len(filters.IgnoredLabels))
			}
			for i, label := range tc.expectedFilter.IgnoredLabels {
				if i >= len(filters.IgnoredLabels) || filters.IgnoredLabels[i] != label {
					t.Errorf("Expected ignored-labels[%d] '%s', got '%s'", i, label, filters.IgnoredLabels[i])
				}
			}

			if len(filters.IgnoredTerms) != len(tc.expectedFilter.IgnoredTerms) {
				t.Errorf("Expected ignored-terms length %d, got %d", len(tc.expectedFilter.IgnoredTerms), len(filters.IgnoredTerms))
			}
			for i, term := range tc.expectedFilter.IgnoredTerms {
				if i >= len(filters.IgnoredTerms) || filters.IgnoredTerms[i] != term {
					t.Errorf("Expected ignored-terms[%d] '%s', got '%s'", i, term, filters.IgnoredTerms[i])
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
			input:          `{"authors": ["alice"], "ignored-authors": ["bob"]}`,
			expectedErrMsg: "cannot use both authors and ignored-authors filters at the same time",
		},
		{
			name:           "conflicting labels",
			input:          `{"labels": ["feature"], "ignored-labels": ["feature"]}`,
			expectedErrMsg: "labels filter cannot contain labels that are in ignored-labels filter",
		},
		{
			name:           "empty string in ignored-terms",
			input:          `{"ignored-terms": ["valid term", ""]}`,
			expectedErrMsg: "ignored-terms cannot contain empty strings",
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
			input: `repo1: {"authors": ["alice"]}; repo2: {"ignored-labels": ["wip"]}`,
			expectedFilters: map[string]config.Filters{
				"repo1": {
					Authors: []string{"alice"},
				},
				"repo2": {
					IgnoredLabels: []string{"wip"},
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
				if len(actualFilter.IgnoredLabels) != len(expectedFilter.IgnoredLabels) {
					t.Errorf("Repository '%s': expected ignored-labels length %d, got %d", repo, len(expectedFilter.IgnoredLabels), len(actualFilter.IgnoredLabels))
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
			input:          `repo1: {"authors": ["alice"], "ignored-authors": ["bob"]}`,
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
			name: "ignored-authors only",
			filter: config.Filters{
				IgnoredAuthors: []string{"charlie"},
			},
			shouldBeValid: true,
		},
		{
			name: "labels and ignored-labels without overlap",
			filter: config.Filters{
				Labels:        []string{"feature"},
				IgnoredLabels: []string{"wip"},
			},
			shouldBeValid: true,
		},
		{
			name: "ignored-terms only",
			filter: config.Filters{
				IgnoredTerms: []string{"Release v1.0", "Automated Update"},
			},
			shouldBeValid: true,
		},
		{
			name: "conflicting authors",
			filter: config.Filters{
				Authors:        []string{"alice"},
				IgnoredAuthors: []string{"bob"},
			},
			shouldBeValid:  false,
			expectedErrMsg: "cannot use both authors and ignored-authors filters at the same time",
		},
		{
			name: "overlapping labels",
			filter: config.Filters{
				Labels:        []string{"feature", "bug"},
				IgnoredLabels: []string{"feature", "wip"},
			},
			shouldBeValid:  false,
			expectedErrMsg: "labels filter cannot contain labels that are in ignored-labels filter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)

			var jsonStr string
			if len(tc.filter.Authors) > 0 || len(tc.filter.IgnoredAuthors) > 0 ||
				len(tc.filter.Labels) > 0 || len(tc.filter.IgnoredLabels) > 0 || len(tc.filter.IgnoredTerms) > 0 {
				parts := []string{}
				if len(tc.filter.Authors) > 0 {
					parts = append(parts, `"authors": ["`+strings.Join(tc.filter.Authors, `", "`)+`"]`)
				}
				if len(tc.filter.IgnoredAuthors) > 0 {
					parts = append(parts, `"ignored-authors": ["`+strings.Join(tc.filter.IgnoredAuthors, `", "`)+`"]`)
				}
				if len(tc.filter.Labels) > 0 {
					parts = append(parts, `"labels": ["`+strings.Join(tc.filter.Labels, `", "`)+`"]`)
				}
				if len(tc.filter.IgnoredLabels) > 0 {
					parts = append(parts, `"ignored-labels": ["`+strings.Join(tc.filter.IgnoredLabels, `", "`)+`"]`)
				}
				if len(tc.filter.IgnoredTerms) > 0 {
					parts = append(parts, `"ignored-terms": ["`+strings.Join(tc.filter.IgnoredTerms, `", "`)+`"]`)
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
