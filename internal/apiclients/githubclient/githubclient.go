// Package githubclient provides GitHub API integration for fetching PR data.
// It handles concurrent repository queries, review data fetching, and applies
// repository-specific and global filters to PRs.
package githubclient

import (
	"context"
	"fmt"
	"log"

	"time"

	"github.com/google/go-github/v72/github"
	"github.com/hellej/pr-slack-reminder-action/internal/config"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
	"golang.org/x/sync/errgroup"
)

type Client interface {
	FetchOpenPRs(
		ctx context.Context,
		repositories []models.Repository,
		getFiltersForRepository func(repo models.Repository) config.Filters,
	) ([]PR, error)
}

type GithubPullRequestsService interface {
	List(
		ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions,
	) (
		[]*github.PullRequest, *github.Response, error,
	)
}

type GithubIssueService interface {
	ListIssueTimeline(
		ctx context.Context, owner string, repo string, number int, opts *github.ListOptions,
	) (
		[]*github.Timeline, *github.Response, error,
	)
}

func NewClient(prsService GithubPullRequestsService, issuesService GithubIssueService) Client {
	return &client{prsService: prsService, issuesService: issuesService}
}

func GetAuthenticatedClient(token string) Client {
	ghClient := github.NewClient(nil).WithAuthToken(token)
	return NewClient(ghClient.PullRequests, ghClient.Issues)
}

type client struct {
	prsService    GithubPullRequestsService
	issuesService GithubIssueService
}

// DefaultGitHubAPIConcurrencyLimit caps concurrent repository fetches to avoid
// creating excessive simultaneous GitHub API calls when many repositories are configured.
// Exported to allow tests (and potential future configuration) to reference it.
const DefaultGitHubAPIConcurrencyLimit = 5

// Per-call timeout defaults. Overridable in tests.
var PullRequestListTimeout = 10 * time.Second
var TimelineFetchTimeout = 10 * time.Second

// Returns an error if fetching PRs from any repository fails (and cancels the other requests).
func (c *client) FetchOpenPRs(
	ctx context.Context,
	repositories []models.Repository,
	getFiltersForRepository func(repo models.Repository) config.Filters,
) ([]PR, error) {
	log.Printf("Fetching open pull requests for repositories: %v", repositories)

	listGroup, listCtx := errgroup.WithContext(ctx)
	listGroup.SetLimit(DefaultGitHubAPIConcurrencyLimit)
	prResults := make([]PRsOfRepoResult, len(repositories))

	for i, repo := range repositories {
		i, repo := i, repo // https://golang.org/doc/faq#closures_and_goroutines
		listGroup.Go(func() error {
			res, err := c.fetchOpenPRsForRepository(listCtx, repo)
			if err == nil {
				prResults[i] = res
			}
			return err
		})
	}
	if err := listGroup.Wait(); err != nil {
		return nil, err
	}

	filteredResults := utilities.Map(
		prResults,
		func(r PRsOfRepoResult) PRsOfRepoResult {
			return PRsOfRepoResult{
				prs:        utilities.Filter(r.prs, getPRFilterFunc(getFiltersForRepository(r.repository))),
				repository: r.repository,
			}
		},
	)
	logFoundPRs(filteredResults)

	prs, err := c.addReviewerInfoToPRs(
		ctx,
		utilities.Filter(
			filteredResults,
			func(r PRsOfRepoResult) bool {
				return r.GetPRCount() > 0
			},
		),
	)
	return prs, err
}

func getPRFilterFunc(filters config.Filters) func(pr *github.PullRequest) bool {
	return func(pr *github.PullRequest) bool {
		return !pr.GetDraft() && includePR(pr, filters)
	}
}

func (c *client) fetchOpenPRsForRepository(
	ctx context.Context, repo models.Repository,
) (PRsOfRepoResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, PullRequestListTimeout)
	defer cancel()
	prs, response, err := c.prsService.List(
		callCtx, repo.Owner, repo.Name, &github.PullRequestListOptions{ListOptions: github.ListOptions{PerPage: 100}},
	)
	if err == nil {
		return PRsOfRepoResult{
			prs:        prs,
			repository: repo,
		}, nil
	}
	if response != nil && response.StatusCode == 404 {
		return PRsOfRepoResult{
				prs:        nil,
				repository: repo,
			},
			fmt.Errorf(
				"repository %s/%s not found - check the repository name and permissions",
				repo.Owner,
				repo.Name,
			)
	}
	return PRsOfRepoResult{
			prs:        nil,
			repository: repo,
		},
		fmt.Errorf(
			"error fetching pull requests from %s/%s: %w", repo.Owner, repo.Name, err,
		)
}

func logFoundPRs(prResults []PRsOfRepoResult) {
	for _, result := range prResults {
		log.Printf("Found %d open pull requests in repository %s:", len(result.prs), result.repository)
		for _, pr := range result.prs {
			log.Printf("  #%v: %s \"%s\"", *pr.Number, pr.GetHTMLURL(), pr.GetTitle())
		}
	}
}

// Fetches review and comment data for the given PRs and returns enriched PR data.
// Returns all PRs even if fetching review data for some PRs fails (those will just be missing reviewer info then).
func (c *client) addReviewerInfoToPRs(ctx context.Context, prResults []PRsOfRepoResult) ([]PR, error) {
	log.Printf("Fetching pull request timelines for PRs")

	totalPRCount := 0
	for _, result := range prResults {
		totalPRCount += result.GetPRCount()
	}

	timelineGroup, timelineCtx := errgroup.WithContext(ctx)
	timelineGroup.SetLimit(DefaultGitHubAPIConcurrencyLimit)
	resultChannel := make(chan FetchTimelineResult, totalPRCount)

	for _, result := range prResults {
		for _, pullRequest := range result.prs {
			repo := result.repository
			pr := pullRequest
			timelineGroup.Go(func() error {
				callCtx, cancel := context.WithTimeout(timelineCtx, TimelineFetchTimeout)
				defer cancel()
				timelineEvents, err := fetchPRTimeline(
					callCtx, c.issuesService, repo.Owner, repo.Name, *pr.Number,
				)
				fetchTimelineResult := FetchTimelineResult{
					pr:             pr,
					timelineEvents: timelineEvents,
					repository:     repo,
				}
				if err != nil {
					fetchTimelineResult.err = err
				}
				resultChannel <- fetchTimelineResult
				return nil
			})
		}
	}

	if err := timelineGroup.Wait(); err != nil {
		return nil, err
	}
	close(resultChannel)

	allPRs := []PR{}
	for result := range resultChannel {
		result.printResult()
		allPRs = append(allPRs, result.asPR())
	}
	return allPRs, nil
}

const timelineMaximumPages = 4

func fetchPRTimeline(
	ctx context.Context,
	issuesService GithubIssueService,
	owner, repo string,
	number int,
) ([]*github.Timeline, error) {
	events := []*github.Timeline{}
	opts := &github.ListOptions{PerPage: 100}
	pagesFetched := 0

	for {
		timelineEvents, response, err := issuesService.ListIssueTimeline(ctx, owner, repo, number, opts)

		if err != nil {
			statusText := ""
			if response != nil && response.Status != "" {
				statusText = " status=" + response.Status
			}
			return nil, fmt.Errorf(
				"error fetching reviews for pull request %s/%s/%d%s: %w",
				owner,
				repo,
				number,
				statusText,
				err,
			)
		}

		events = append(events, timelineEvents...)
		pagesFetched++

		if response == nil || response.NextPage == 0 || pagesFetched >= timelineMaximumPages {
			break
		}
		opts.Page = response.NextPage
	}
	return events, nil
}
