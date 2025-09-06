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

func TestMultipleRepositories(t *testing.T) {
	h := newConfigTestHelpers(t)
	h.setupMinimalValidConfig(MinimalConfigOptions{
		SkipGithubRepository: true, // We'll set our own repository
	})
	h.setEnv(config.EnvGithubRepository, "default-org/default-repo")
	h.setInputList(config.InputGithubRepositories, []string{
		"org1/repo1",
		"org2/repo2",
		"org3/repo3",
	})

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedRepos := []config.Repository{
		{Path: "org1/repo1", Owner: "org1", Name: "repo1"},
		{Path: "org2/repo2", Owner: "org2", Name: "repo2"},
		{Path: "org3/repo3", Owner: "org3", Name: "repo3"},
	}

	if len(cfg.Repositories) != len(expectedRepos) {
		t.Fatalf("Expected %d repositories, got %d", len(expectedRepos), len(cfg.Repositories))
	}

	for i, expectedRepo := range expectedRepos {
		repo := cfg.Repositories[i]
		if repo.Path != expectedRepo.Path {
			t.Errorf("Repository %d: expected path '%s', got '%s'", i, expectedRepo.Path, repo.Path)
		}
		if repo.Owner != expectedRepo.Owner {
			t.Errorf("Repository %d: expected owner '%s', got '%s'", i, expectedRepo.Owner, repo.Owner)
		}
		if repo.Name != expectedRepo.Name {
			t.Errorf("Repository %d: expected name '%s', got '%s'", i, expectedRepo.Name, repo.Name)
		}
	}
}

func TestMultipleRepositories_WithInvalid(t *testing.T) {
	h := newConfigTestHelpers(t)
	h.setupMinimalValidConfig(MinimalConfigOptions{
		SkipGithubRepository: true, // We'll set our own repository
	})
	h.setEnv(config.EnvGithubRepository, "default-org/default-repo")
	h.setInputList(config.InputGithubRepositories, []string{
		"org1/repo1",
		"invalid-repo", // This should cause an error
		"org3/repo3",
	})

	_, err := config.GetConfig()
	if err == nil {
		t.Fatalf("Expected error due to invalid repository, got nil")
	}
	expectedErrMsg := "invalid repositories input: invalid owner/repository format: invalid-repo"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestRepositoriesFallbackToDefault(t *testing.T) {
	// When github-repositories is not set, should fall back to GITHUB_REPOSITORY
	h := newConfigTestHelpers(t)
	h.setupMinimalValidConfig(MinimalConfigOptions{SkipGithubRepository: true})
	h.setEnv(config.EnvGithubRepository, "fallback-org/fallback-repo")

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(cfg.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(cfg.Repositories))
	}

	repo := cfg.Repositories[0]
	if repo.Path != "fallback-org/fallback-repo" {
		t.Errorf("Expected repository path 'fallback-org/fallback-repo', got '%s'", repo.Path)
	}
	if repo.Owner != "fallback-org" {
		t.Errorf("Expected repository owner 'fallback-org', got '%s'", repo.Owner)
	}
	if repo.Name != "fallback-repo" {
		t.Errorf("Expected repository name 'fallback-repo', got '%s'", repo.Name)
	}
}
