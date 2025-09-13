// Package config handles GitHub Action input parsing and validation.
// It converts environment variables to structured configuration with
// support for repository-specific filters, user mappings, and content settings.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/hellej/pr-slack-reminder-action/internal/config/inputhelpers"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
)

const (
	EnvGithubRepository string = "GITHUB_REPOSITORY"

	InputGithubRepositories          string = "github-repositories"
	InputGithubToken                 string = "github-token"
	InputSlackBotToken               string = "slack-bot-token"
	InputSlackChannelName            string = "slack-channel-name"
	InputSlackChannelID              string = "slack-channel-id"
	InputSlackUserIdByGitHubUsername string = "github-user-slack-user-id-mapping"
	InputNoPRsMessage                string = "no-prs-message"
	InputPRListHeading               string = "main-list-heading"
	InputOldPRThresholdHours         string = "old-pr-threshold-hours"
	InputGlobalFilters               string = "filters"
	InputRepositoryFilters           string = "repository-filters"
	InputRepositoryPrefixes          string = "repository-prefixes"

	MaxRepositories int = 50
)

type Config struct {
	GithubToken   string
	SlackBotToken string

	repository   string
	Repositories []Repository

	SlackChannelName string
	SlackChannelID   string

	GlobalFilters     Filters
	RepositoryFilters map[string]Filters
	ContentInputs     ContentInputs
}

type ContentInputs struct {
	NoPRsMessage                string
	PRListHeading               string
	OldPRThresholdHours         int
	RepositoryPrefixes          map[string]string
	SlackUserIdByGitHubUsername map[string]string
}

func (c Config) Print() {
	copy := c
	if copy.GithubToken != "" {
		copy.GithubToken = "XXXXX"
	}
	if copy.SlackBotToken != "" {
		copy.SlackBotToken = "XXXXX"
	}
	asJson, _ := json.MarshalIndent(copy, "", "  ")
	log.Print("Configuration:")
	log.Println(string(asJson))
}

func GetConfig() (Config, error) {
	repository, err1 := inputhelpers.GetEnvRequired(EnvGithubRepository)
	githubToken, err2 := inputhelpers.GetInputRequired(InputGithubToken)
	slackToken, err3 := inputhelpers.GetInputRequired(InputSlackBotToken)
	mainListHeading, err4 := inputhelpers.GetInputRequired(InputPRListHeading)
	oldPRsThresholdHours, err5 := inputhelpers.GetInputInt(InputOldPRThresholdHours)
	slackUserIdByGitHubUsername, err6 := inputhelpers.GetInputMapping(InputSlackUserIdByGitHubUsername)
	globalFilters, err7 := GetGlobalFiltersFromInput(InputGlobalFilters)
	repositoryFilters, err8 := GetRepositoryFiltersFromInput(InputRepositoryFilters)
	repositoryPrefixes, err9 := inputhelpers.GetInputMapping(InputRepositoryPrefixes)

	if err := selectNonNilError(err1, err2, err3, err4, err5, err6, err7, err8, err9); err != nil {
		return Config{}, err
	}

	repositoryPaths := inputhelpers.GetInputList(InputGithubRepositories)
	if len(repositoryPaths) == 0 {
		repositoryPaths = []string{repository}
	}

	repositories := make([]Repository, len(repositoryPaths))
	for i, repoPath := range repositoryPaths {
		repo, err := parseRepository(repoPath)
		if err != nil {
			return Config{}, fmt.Errorf("invalid repositories input: %v", err)
		}
		repositories[i] = repo
	}

	config := Config{
		repository:       repository,
		Repositories:     repositories,
		GithubToken:      githubToken,
		SlackBotToken:    slackToken,
		SlackChannelName: inputhelpers.GetInput(InputSlackChannelName),
		SlackChannelID:   inputhelpers.GetInput(InputSlackChannelID),
		ContentInputs: ContentInputs{
			SlackUserIdByGitHubUsername: slackUserIdByGitHubUsername,
			NoPRsMessage:                inputhelpers.GetInput(InputNoPRsMessage),
			PRListHeading:               mainListHeading,
			OldPRThresholdHours:         oldPRsThresholdHours,
			RepositoryPrefixes:          repositoryPrefixes,
		},
		GlobalFilters:     globalFilters,
		RepositoryFilters: repositoryFilters,
	}

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// validate performs post-construction validation of business rules for Config.
// It validates repository limits, Slack channel requirements, and repository consistency.
func (c Config) validate() error {
	if c.SlackChannelID == "" && c.SlackChannelName == "" {
		return fmt.Errorf("either %s or %s must be set", InputSlackChannelID, InputSlackChannelName)
	}
	if len(c.Repositories) > MaxRepositories {
		return fmt.Errorf("too many repositories: maximum of %d repositories allowed, got %d", MaxRepositories, len(c.Repositories))
	}
	if err := c.validateRepositoryNames(); err != nil {
		return err
	}

	return nil
}

func (c Config) validateRepositoryNames() error {
	// Check for duplicates
	repositoryPaths := make(map[string]bool, len(c.Repositories))
	for _, repo := range c.Repositories {
		if repositoryPaths[repo.Path] {
			return fmt.Errorf("duplicate repository '%s' found in github-repositories", repo.Path)
		}
		repositoryPaths[repo.Path] = true
	}

	// Check that repository filters and prefixes reference valid repositories
	repositoryNames := make(map[string]bool, len(c.Repositories))
	for _, repo := range c.Repositories {
		repositoryNames[repo.Name] = true
	}

	for repoName := range c.RepositoryFilters {
		if !repositoryNames[repoName] {
			return fmt.Errorf(
				"repository-filters contains entry for '%s' which is not in github-repositories",
				repoName,
			)
		}
	}

	for repoName := range c.ContentInputs.RepositoryPrefixes {
		if !repositoryNames[repoName] {
			return fmt.Errorf(
				"repository-prefixes contains entry for '%s' which is not in github-repositories",
				repoName,
			)
		}
	}

	// check for ambiguous repository identifiers (same repo name with different owners)
	if len(c.Repositories) > 1 {
		if err := checkForAmbiguousRepositoryNames(c.Repositories, c.RepositoryFilters, "repository-filters"); err != nil {
			return err
		}
		if err := checkForAmbiguousRepositoryNames(c.Repositories, c.ContentInputs.RepositoryPrefixes, "repository-prefixes"); err != nil {
			return err
		}
	}

	return nil
}

func selectNonNilError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// checkForAmbiguousRepositoryNames checks if any repository name appears multiple times
// across different owners, which would make repository-specific configurations ambiguous.
func checkForAmbiguousRepositoryNames[V any](repositories []Repository, repoMapping map[string]V, inputName string) error {
	for repoName := range repoMapping {
		if len(
			slices.Collect(
				utilities.Filter(repositories, func(r Repository) bool {
					return r.Name == repoName
				}),
			),
		) > 1 {
			return fmt.Errorf(
				"%s contains ambiguous entry for '%s' which matches "+
					"multiple repositories (needs owner/repo format)",
				inputName,
				repoName,
			)
		}
	}
	return nil
}
