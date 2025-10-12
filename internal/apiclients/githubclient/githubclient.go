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
	"golang.org/x/sync/errgroup"
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

// DefaultRepoFetchConcurrencyLimit caps concurrent repository fetches to avoid
// creating excessive simultaneous GitHub API calls when many repositories are configured.
// Exported to allow tests (and potential future configuration) to reference it.
const DefaultRepoFetchConcurrencyLimit = 5

// Returns an error if fetching PRs from any repository fails (and cancels the other requests).
func (c *client) FetchOpenPRs(
	repositories []models.Repository,
	getFiltersForRepository func(repo models.Repository) config.Filters,
) ([]PR, error) {
	log.Printf("Fetching open pull requests for repositories: %v", repositories)

	eg, ctx := errgroup.WithContext(context.Background())
	eg.SetLimit(DefaultRepoFetchConcurrencyLimit)
	prResults := make([]PRsOfRepoResult, len(repositories))

	for i, repo := range repositories {
		i, repo := i, repo // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			res, err := c.fetchOpenPRsForRepository(ctx, repo)
			if err == nil {
				prResults[i] = res
			}
			return err
		})
	}
	if err := eg.Wait(); err != nil {
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

	prs := c.addReviewerInfoToPRs(
		utilities.Filter(
			filteredResults,
			func(r PRsOfRepoResult) bool {
				return r.GetPRCount() > 0
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
) (PRsOfRepoResult, error) {
	prs, response, err := c.prsService.List(
		ctx, repo.Owner, repo.Name, &github.PullRequestListOptions{ListOptions: github.ListOptions{PerPage: 50}},
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
			"error fetching pull requests from %s/%s: %v", repo.Owner, repo.Name, err,
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

func (c *client) addReviewerInfoToPRs(prResults []PRsOfRepoResult) []PR {
	log.Printf("Fetching pull request timelines for PRs")

	totalPRCount := 0
	for _, result := range prResults {
		totalPRCount += result.GetPRCount()
	}

	resultChannel := make(chan FetchTimelineResult, totalPRCount)
	var wg sync.WaitGroup

	for _, result := range prResults {
		for _, pullRequest := range result.prs {
			wg.Add(1)
			go func(repo models.Repository, pr *github.PullRequest) {
				defer wg.Done()
				timelineEvents, response, err := c.issuesService.ListIssueTimeline(
					context.Background(), repo.Owner, repo.Name, *pr.Number, &github.ListOptions{PerPage: 100},
				)
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
				prWithTimeline := FetchTimelineResult{
					pr:             pr,
					timelineEvents: timelineEvents,
					repository:     repo,
					err:            err,
				}
				resultChannel <- prWithTimeline

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
