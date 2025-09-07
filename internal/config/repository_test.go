package config_test

import (
	"strings"
	"testing"

	"github.com/hellej/pr-slack-reminder-action/internal/config"
)

func TestRepositoryString(t *testing.T) {
	repo := config.Repository{
		Path:  "test-org/test-repo",
		Owner: "test-org",
		Name:  "test-repo",
	}

	expected := "test-org/test-repo"
	if repo.String() != expected {
		t.Errorf("Expected repository string '%s', got '%s'", expected, repo.String())
	}
}

func TestParseRepository_Valid(t *testing.T) {
	testCases := []struct {
		name           string
		repositoryPath string
		expectedRepo   config.Repository
	}{
		{
			name:           "standard repository path",
			repositoryPath: "octocat/Hello-World",
			expectedRepo: config.Repository{
				Path:  "octocat/Hello-World",
				Owner: "octocat",
				Name:  "Hello-World",
			},
		},
		{
			name:           "repository with numbers",
			repositoryPath: "org123/repo456",
			expectedRepo: config.Repository{
				Path:  "org123/repo456",
				Owner: "org123",
				Name:  "repo456",
			},
		},
		{
			name:           "repository with hyphens and underscores",
			repositoryPath: "my-org/my_repo-name",
			expectedRepo: config.Repository{
				Path:  "my-org/my_repo-name",
				Owner: "my-org",
				Name:  "my_repo-name",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setupMinimalValidConfig(MinimalConfigOptions{
				SkipGithubRepository: true, // We'll set our own repository
			})
			h.setEnv(config.EnvGithubRepository, tc.repositoryPath)

			cfg, err := config.GetConfig()
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(cfg.Repositories) != 1 {
				t.Fatalf("Expected 1 repository, got %d", len(cfg.Repositories))
			}

			repo := cfg.Repositories[0]
			if repo.Path != tc.expectedRepo.Path {
				t.Errorf("Expected repository path '%s', got '%s'", tc.expectedRepo.Path, repo.Path)
			}
			if repo.Owner != tc.expectedRepo.Owner {
				t.Errorf("Expected repository owner '%s', got '%s'", tc.expectedRepo.Owner, repo.Owner)
			}
			if repo.Name != tc.expectedRepo.Name {
				t.Errorf("Expected repository name '%s', got '%s'", tc.expectedRepo.Name, repo.Name)
			}
		})
	}
}

func TestParseRepository_Invalid(t *testing.T) {
	testCases := []struct {
		name           string
		repositoryPath string
		expectedErrMsg string
	}{
		{
			name:           "no slash",
			repositoryPath: "invalid-repo-format",
			expectedErrMsg: "invalid owner/repository format: invalid-repo-format",
		},
		{
			name:           "too many slashes",
			repositoryPath: "org/repo/extra",
			expectedErrMsg: "invalid owner/repository format: org/repo/extra",
		},
		{
			name:           "empty owner",
			repositoryPath: "/repo-name",
			expectedErrMsg: "owner or repository name cannot be empty in: /repo-name",
		},
		{
			name:           "empty repository name",
			repositoryPath: "org-name/",
			expectedErrMsg: "owner or repository name cannot be empty in: org-name/",
		},
		{
			name:           "both empty",
			repositoryPath: "/",
			expectedErrMsg: "owner or repository name cannot be empty in: /",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setupMinimalValidConfig(MinimalConfigOptions{
				SkipGithubRepository: true, // We'll set our own repository
			})
			h.setEnv(config.EnvGithubRepository, tc.repositoryPath)

			_, err := config.GetConfig()
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErrMsg, err.Error())
			}
		})
	}
}
