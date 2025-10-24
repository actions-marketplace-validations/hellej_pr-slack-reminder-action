package mockgithubclient

import (
	"context"
	"net/http"

	"github.com/google/go-github/v72/github"
	"github.com/hellej/pr-slack-reminder-action/internal/apiclients/githubclient"
)

func MakeMockGitHubClientGetter(
	prs []*github.PullRequest,
	prsByRepo map[string][]*github.PullRequest,
	listPRsResponseStatus int,
	listPRsErr error,
	reviewsByPRNumber map[int][]*github.PullRequestReview,
) func(token string) githubclient.Client {
	return func(token string) githubclient.Client {
		mockPRService := &mockPullRequestsService{
			mockPRs:               prs,
			mockPRsByRepo:         prsByRepo,
			mockReviewsByPRNumber: reviewsByPRNumber,
			mockResponse: &github.Response{
				Response: &http.Response{
					StatusCode: listPRsResponseStatus,
				},
			},
			mockError: listPRsErr,
		}
		return githubclient.NewClient(mockPRService)
	}
}

func NewReview(id int64, state, login, name, body string, userType ...string) *github.PullRequestReview {
	var t *string
	if len(userType) > 0 && userType[0] != "" {
		t = github.Ptr(userType[0])
	}
	var b *string
	if body != "" {
		b = github.Ptr(body)
	}
	return &github.PullRequestReview{
		ID:   github.Ptr(id),
		Body: b,
		User: &github.User{
			Login: github.Ptr(login),
			Name:  github.Ptr(name),
			Type:  t,
		},
		State: github.Ptr(state),
	}
}

type mockPullRequestsService struct {
	mockPRs               []*github.PullRequest
	mockPRsByRepo         map[string][]*github.PullRequest
	mockReviewsByPRNumber map[int][]*github.PullRequestReview
	mockResponse          *github.Response
	mockError             error
}

func (m *mockPullRequestsService) List(
	ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	if m.mockPRsByRepo != nil {
		return m.mockPRsByRepo[repo], m.mockResponse, m.mockError
	}
	return m.mockPRs, m.mockResponse, m.mockError
}

func (m *mockPullRequestsService) ListReviews(
	ctx context.Context, owner string, repo string, number int, opts *github.ListOptions,
) ([]*github.PullRequestReview, *github.Response, error) {
	reviews := m.mockReviewsByPRNumber[number]
	return reviews, m.mockResponse, m.mockError
}
