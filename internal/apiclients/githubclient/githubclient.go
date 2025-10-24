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
	ListReviews(
		ctx context.Context, owner string, repo string, number int, opts *github.ListOptions,
	) (
		[]*github.PullRequestReview, *github.Response, error,
	)
}

func NewClient(prsService GithubPullRequestsService) Client {
	return &client{prsService: prsService}
}

func GetAuthenticatedClient(token string) Client {
	ghClient := github.NewClient(nil).WithAuthToken(token)
	return NewClient(ghClient.PullRequests)
}

type client struct {
	prsService GithubPullRequestsService
}

// DefaultGitHubAPIConcurrencyLimit caps concurrent repository fetches to avoid
// creating excessive simultaneous GitHub API calls when many repositories are configured.
// Exported to allow tests (and potential future configuration) to reference it.
const DefaultGitHubAPIConcurrencyLimit = 5

// Per-call timeout defaults. Overridable in tests.
var PullRequestListTimeout = 10 * time.Second
var ReviewsFetchTimeout = 10 * time.Second

// Returns an error if fetching PRs from any repository fails (and cancels the other requests).
func (c *client) FetchOpenPRs(
	ctx context.Context,
	repositories []models.Repository,
	getFiltersForRepository func(repo models.Repository) config.Filters,
) ([]PR, error) {
	log.Printf("Fetching open pull requests for repositories: %v", repositories)

	listGroup, listCtx := errgroup.WithContext(ctx)
	listGroup.SetLimit(DefaultGitHubAPIConcurrencyLimit)
	prResultSlices := make([][]PRResult, len(repositories))

	for i, repo := range repositories {
		i, repo := i, repo // https://golang.org/doc/faq#closures_and_goroutines
		listGroup.Go(func() error {
			res, err := c.fetchOpenPRsForRepository(listCtx, repo)
			if err == nil {
				prResultSlices[i] = res
			}
			return err
		})
	}
	if err := listGroup.Wait(); err != nil {
		return nil, err
	}

	prResults := utilities.Filter(
		utilities.FlatMap(prResultSlices),
		getPRFilterFunc(getFiltersForRepository),
	)
	logFoundPRs(prResults)

	return c.addReviewerInfoToPRs(ctx, prResults)
}

func getPRFilterFunc(
	getFiltersForRepository func(repo models.Repository) config.Filters,
) func(result PRResult) bool {
	return func(result PRResult) bool {
		return !result.pr.GetDraft() && includePR(result.pr, getFiltersForRepository(result.repository))
	}
}

func (c *client) fetchOpenPRsForRepository(
	ctx context.Context, repo models.Repository,
) ([]PRResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, PullRequestListTimeout)
	defer cancel()
	prs, response, err := c.prsService.List(
		callCtx, repo.Owner, repo.Name, &github.PullRequestListOptions{ListOptions: github.ListOptions{PerPage: 100}},
	)
	if err == nil {
		return utilities.Map(prs, getPRResultMapper(repo)), nil
	}
	if response != nil && response.StatusCode == 404 {
		return nil, fmt.Errorf(
			"repository %s/%s not found - check the repository name and permissions",
			repo.Owner,
			repo.Name,
		)
	}
	return nil, fmt.Errorf(
		"error fetching pull requests from %s/%s: %w", repo.Owner, repo.Name, err,
	)
}

func getPRResultMapper(repo models.Repository) func(pr *github.PullRequest) PRResult {
	return func(pr *github.PullRequest) PRResult {
		return PRResult{
			pr:         pr,
			repository: repo,
		}
	}
}

func logFoundPRs(prResults []PRResult) {
	log.Printf("Found %d open pull requests:", len(prResults))
	for _, result := range prResults {
		log.Printf("%s/%v", result.repository.GetPath(), *result.pr.Number)
	}
}

// Fetches review and comment data for the given PRs and returns enriched PR data.
// Returns all PRs even if fetching review data for some PRs fails (those will just be missing reviewer info then).
func (c *client) addReviewerInfoToPRs(ctx context.Context, prResults []PRResult) ([]PR, error) {
	log.Printf("\nFetching pull request reviews for PRs")

	reviewsGroup, reviewsCtx := errgroup.WithContext(ctx)
	reviewsGroup.SetLimit(DefaultGitHubAPIConcurrencyLimit)
	resultChannel := make(chan FetchReviewsResult, len(prResults))

	for _, result := range prResults {
		repo := result.repository
		pr := result.pr
		reviewsGroup.Go(func() error {
			callCtx, cancel := context.WithTimeout(reviewsCtx, ReviewsFetchTimeout)
			defer cancel()
			reviews, err := fetchPRReviews(
				callCtx, c.prsService, repo.Owner, repo.Name, *pr.Number,
			)
			fetchReviewsResult := FetchReviewsResult{
				pr:         pr,
				reviews:    reviews,
				repository: repo,
			}
			if err != nil {
				fetchReviewsResult.err = err
			}
			resultChannel <- fetchReviewsResult
			return nil
		})
	}

	if err := reviewsGroup.Wait(); err != nil {
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

const reviewsMaximumPages = 2

func fetchPRReviews(
	ctx context.Context,
	prsService GithubPullRequestsService,
	owner, repo string,
	number int,
) ([]*github.PullRequestReview, error) {
	reviews := []*github.PullRequestReview{}
	opts := &github.ListOptions{PerPage: 100}
	pagesFetched := 0

	for {
		reviewsPage, response, err := prsService.ListReviews(ctx, owner, repo, number, opts)

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

		reviews = append(reviews, reviewsPage...)
		pagesFetched++

		if response == nil || response.NextPage == 0 || pagesFetched >= reviewsMaximumPages {
			break
		}
		opts.Page = response.NextPage
	}
	return reviews, nil
}
