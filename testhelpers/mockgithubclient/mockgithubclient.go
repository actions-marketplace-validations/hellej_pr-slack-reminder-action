package mockgithubclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/go-github/v78/github"
	"github.com/hellej/pr-slack-reminder-action/internal/apiclients/githubclient"
)

func MakeMockGitHubClientGetter(
	prsByNumber map[int]*github.PullRequest,
	errByPRNumber map[int]error,
	prs []*github.PullRequest,
	prsByRepo map[string][]*github.PullRequest,
	listPRsResponseStatus int,
	reviewsByPRNumber map[int][]*github.PullRequestReview,
	commentsByPRNumber map[int][]*github.PullRequestComment,
	prServiceError error,
) func(token string) githubclient.Client {
	return func(token string) githubclient.Client {
		mockPRService := &mockPullRequestService{
			prsByNumber:        prsByNumber,
			errorByPRNumber:    errByPRNumber,
			prs:                prs,
			prsByRepo:          prsByRepo,
			reviewsByPRNumber:  reviewsByPRNumber,
			commentsByPRNumber: commentsByPRNumber,
			response: &github.Response{
				Response: &http.Response{
					StatusCode: listPRsResponseStatus,
				},
			},
			err: prServiceError,
		}
		mockIssueService := &mockIssueService{
			mockTimelineCommentsByPRNumber: map[int][]*github.IssueComment{},
			response: &github.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			},
			err: nil,
		}
		mockHTTPClient := &mockHTTPClient{
			response: &http.Response{
				StatusCode: 200,
			},
			err: nil,
		}
		mockActionsService := &mockActionsService{
			response: &github.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			},
			err: nil,
		}
		return githubclient.NewClient(mockHTTPClient, mockPRService, mockIssueService, mockActionsService)
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

func NewComment(id int64, login, name, body string, userType ...string) *github.PullRequestComment {
	var t *string
	if len(userType) > 0 && userType[0] != "" {
		t = github.Ptr(userType[0])
	}
	var b *string
	if body != "" {
		b = github.Ptr(body)
	}
	return &github.PullRequestComment{
		ID:   github.Ptr(id),
		Body: b,
		User: &github.User{
			Login: github.Ptr(login),
			Name:  github.Ptr(name),
			Type:  t,
		},
	}
}

type mockPullRequestService struct {
	prsByNumber        map[int]*github.PullRequest
	errorByPRNumber    map[int]error
	prs                []*github.PullRequest
	prsByRepo          map[string][]*github.PullRequest
	reviewsByPRNumber  map[int][]*github.PullRequestReview
	commentsByPRNumber map[int][]*github.PullRequestComment
	response           *github.Response
	err                error
}

func (m *mockPullRequestService) Get(
	ctx context.Context, owner string, repo string, number int,
) (*github.PullRequest, *github.Response, error) {
	if err, ok := m.errorByPRNumber[number]; ok {
		return nil, m.response, err
	}
	if pr, ok := m.prsByNumber[number]; ok {
		return pr, m.response, m.err
	}
	return nil, m.response, m.err
}

func (m *mockPullRequestService) List(
	ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	if m.prsByRepo != nil {
		return m.prsByRepo[repo], m.response, m.err
	}
	return m.prs, m.response, m.err
}

func (m *mockPullRequestService) ListReviews(
	ctx context.Context, owner string, repo string, number int, opts *github.ListOptions,
) ([]*github.PullRequestReview, *github.Response, error) {
	reviews := m.reviewsByPRNumber[number]
	return reviews, m.response, m.err
}

func (m *mockPullRequestService) ListComments(
	ctx context.Context, owner string, repo string, number int, opts *github.PullRequestListCommentsOptions,
) ([]*github.PullRequestComment, *github.Response, error) {
	comments := m.commentsByPRNumber[number]
	return comments, m.response, m.err
}

type mockIssueService struct {
	mockTimelineCommentsByPRNumber map[int][]*github.IssueComment
	response                       *github.Response
	err                            error
}

func (m *mockIssueService) ListComments(
	ctx context.Context, owner string, repo string, number int, opts *github.IssueListCommentsOptions,
) ([]*github.IssueComment, *github.Response, error) {
	comments := m.mockTimelineCommentsByPRNumber[number]
	return comments, m.response, m.err
}

type mockActionsService struct {
	response *github.Response
	err      error
}

func (m *mockActionsService) ListArtifacts(
	ctx context.Context, owner string, repo string, opts *github.ListArtifactsOptions,
) (*github.ArtifactList, *github.Response, error) {
	return &github.ArtifactList{}, m.response, m.err
}

func (m *mockActionsService) DownloadArtifact(
	ctx context.Context, owner, repo string, artifactID int64, maxRedirects int,
) (*url.URL, *github.Response, error) {
	u, _ := url.Parse("https://example.com/mock-download-url")
	return u, m.response, m.err
}

type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Get(url string) (*http.Response, error) {
	return m.response, m.err
}
