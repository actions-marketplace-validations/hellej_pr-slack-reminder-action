// Package githubclient provides GitHub API integration for fetching PR data.
// It handles concurrent repository queries, review data fetching, and applies
// repository-specific and global filters to PRs.
package githubclient

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/go-github/v72/github"
	"github.com/hellej/pr-slack-reminder-action/internal/config"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
)

type Client interface {
	FetchOpenPRs(
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

// Returns an error if fetching PRs from any repository fails (and cancels other requests).
//
// The wait group & cancellation logic could be refactored to use errgroup package for more
// concise implementation. However, the current implementation also serves as learning material
// so we can save the refactoring for later...
func (c *client) FetchOpenPRs(
	repositories []models.Repository,
	getFiltersForRepository func(repo models.Repository) config.Filters,
) ([]PR, error) {
	log.Printf("Fetching open pull requests for repositories: %v", repositories)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	apiResultChannel := make(chan PRsOfRepoResult, len(repositories))

	for _, repo := range repositories {
		wg.Add(1)
		go func(r models.Repository) {
			defer wg.Done()
			apiResult := c.fetchOpenPRsForRepository(ctx, r)
			apiResultChannel <- apiResult
			if apiResult.err != nil {
				cancel()
			}
		}(repo)
	}

	go func() {
		wg.Wait()
		close(apiResultChannel)
	}()

	successfulResults := []PRsOfRepoResult{}
	for result := range apiResultChannel {
		if result.err != nil {
			return nil, result.err
		} else {
			successfulResults = append(successfulResults, result)
		}
	}

	filteredResults := utilities.Map(successfulResults, func(r PRsOfRepoResult) PRsOfRepoResult {
		return PRsOfRepoResult{
			prs:        utilities.Filter(r.prs, getPRFilterFunc(getFiltersForRepository(r.repository))),
			repository: r.repository,
			err:        nil,
		}
	})
	logFoundPRs(filteredResults)

	prs := c.addReviewerInfoToPRs(
		utilities.Filter(
			filteredResults,
			func(r PRsOfRepoResult) bool {
				return len(r.prs) > 0
			},
		),
	)
	return prs, nil
}

func getPRFilterFunc(filters config.Filters) func(pr *github.PullRequest) bool {
	return func(pr *github.PullRequest) bool {
		return !pr.GetDraft() && includePR(pr, filters)
	}
}

func (c *client) fetchOpenPRsForRepository(
	ctx context.Context, repo models.Repository,
) PRsOfRepoResult {
	prs, response, err := c.prsService.List(ctx, repo.Owner, repo.Name, nil)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			return PRsOfRepoResult{
				prs:        nil,
				repository: repo,
				err: fmt.Errorf(
					"repository %s/%s not found - check the repository name and permissions",
					repo.Owner,
					repo.Name,
				)}
		} else {
			return PRsOfRepoResult{
				prs:        nil,
				repository: repo,
				err: fmt.Errorf(
					"error fetching pull requests from %s/%s: %v", repo.Owner, repo.Name, err,
				),
			}
		}
	}
	return PRsOfRepoResult{
		prs:        prs,
		repository: repo,
		err:        nil,
	}
}

func logFoundPRs(prResults []PRsOfRepoResult) {
	for _, result := range prResults {
		log.Printf("Found %d open pull requests in repository %s:", len(result.prs), result.repository)
		for _, pr := range result.prs {
			log.Printf("  #%v: %s \"%s\"", *pr.Number, pr.GetHTMLURL(), pr.GetTitle())
		}
	}
}

func (c *client) addReviewerInfoToPRs(prResults []PRsOfRepoResult) []PR {
	log.Printf("Fetching pull request reviewers for PRs")

	totalPRCount := 0
	for _, result := range prResults {
		totalPRCount += result.GetPRCount()
	}

	resultChannel := make(chan FetchReviewsResult, totalPRCount)
	var wg sync.WaitGroup

	for _, result := range prResults {
		for _, pullRequest := range result.prs {
			wg.Add(1)
			go func(repo models.Repository, pr *github.PullRequest) {
				defer wg.Done()
				reviews, response, err := c.prsService.ListReviews(context.Background(), repo.Owner, repo.Name, *pr.Number, nil)
				if err != nil {
					err = fmt.Errorf(
						"error fetching reviews for pull request %s/%s#%d: %v/%v",
						repo.Owner,
						repo.Name,
						*pr.Number,
						response.Status,
						err,
					)
				}
				prWithReviews := FetchReviewsResult{
					pr:         pr,
					reviews:    reviews,
					repository: repo,
					err:        err,
				}
				resultChannel <- prWithReviews

			}(result.repository, pullRequest)
		}
	}

	go func() {
		wg.Wait()
		close(resultChannel)
	}()

	allPRs := []PR{}
	for result := range resultChannel {
		result.printResult()
		allPRs = append(allPRs, result.asPR())
	}
	return allPRs
}
