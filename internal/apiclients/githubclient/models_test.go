package githubclient

import (
	"testing"

	"github.com/google/go-github/v72/github"
)

func TestCollaboratorGetGitHubName(t *testing.T) {
	tests := []struct {
		name         string
		collaborator Collaborator
		expected     string
	}{
		{
			name:         "has both login and name",
			collaborator: Collaborator{Login: "user1", Name: "User One"},
			expected:     "User One",
		},
		{
			name:         "has login but no name",
			collaborator: Collaborator{Login: "user1", Name: ""},
			expected:     "user1",
		},
		{
			name:         "has login but nil name equivalent",
			collaborator: Collaborator{Login: "user1"},
			expected:     "user1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.collaborator.GetGitHubName()
			if result != tt.expected {
				t.Errorf("GetGitHubName() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestPRsOfRepoResultGetPRCount(t *testing.T) {
	tests := []struct {
		name     string
		result   PRsOfRepoResult
		expected int
	}{
		{
			name: "has PRs",
			result: PRsOfRepoResult{
				prs: []*github.PullRequest{
					{Number: github.Ptr(1)},
					{Number: github.Ptr(2)},
				},
			},
			expected: 2,
		},
		{
			name: "no PRs",
			result: PRsOfRepoResult{
				prs: []*github.PullRequest{},
			},
			expected: 0,
		},
		{
			name: "nil PRs slice",
			result: PRsOfRepoResult{
				prs: nil,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.GetPRCount()
			if result != tt.expected {
				t.Errorf("GetPRCount() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// Note: asPR() is a private method, so we test it indirectly through the public
// FetchOpenPRs method in githubclient_test.go instead of testing implementation details here.
