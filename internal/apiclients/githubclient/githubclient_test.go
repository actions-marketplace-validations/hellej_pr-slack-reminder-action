package githubclient_test

import (
	"testing"

	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v72/github"
	"github.com/hellej/pr-slack-reminder-action/internal/apiclients/githubclient"
	"github.com/hellej/pr-slack-reminder-action/internal/config"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
)

type mockPullRequestsService struct {
	mockPRs      []*github.PullRequest
	mockResponse *github.Response
	mockError    error
}

type mockIssueService struct {
	mockTimelineEventsByPRNumber map[int][]*github.Timeline
	mockResponse                 *github.Response
	mockError                    error
}

func (m *mockPullRequestsService) List(
	ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	return m.mockPRs, m.mockResponse, m.mockError
}

func (m *mockIssueService) ListIssueTimeline(
	ctx context.Context, owner string, repo string, number int, opts *github.ListOptions,
) ([]*github.Timeline, *github.Response, error) {
	reviews := m.mockTimelineEventsByPRNumber[number]
	return reviews, m.mockResponse, m.mockError
}

func NewReview(login, name, state string, userType ...string) *github.Timeline {
	var t *string
	if len(userType) > 0 && userType[0] != "" {
		t = github.Ptr(userType[0])
	}
	return &github.Timeline{
		User: &github.User{
			Login: github.Ptr(login),
			Name:  github.Ptr(name),
			Type:  t,
		},
		State: github.Ptr(state),
		Event: github.Ptr("reviewed"),
	}
}

// multiRepoPRService routes List calls to different mock services based on repo name
type multiRepoPRService struct {
	services map[string]*mockPullRequestsService
}

func (m *multiRepoPRService) List(
	ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	if svc, ok := m.services[repo]; ok {
		return svc.mockPRs, svc.mockResponse, svc.mockError
	}
	return nil, &github.Response{Response: &http.Response{StatusCode: 404}}, fmt.Errorf("unknown repo")
}

func TestGetAuthenticatedClient(t *testing.T) {
	client := githubclient.GetAuthenticatedClient("test-token")
	if client == nil {
		t.Fatal("Expected non-nil client, got nil")
	}
}

func TestFetchOpenPRs(t *testing.T) {
	tests := []struct {
		name                    string
		mockPRs                 []*github.PullRequest
		mockTimelineEvents      map[int][]*github.Timeline
		filters                 config.Filters
		expectedPRCount         int
		expectedApproverLogins  []string
		expectedCommenterLogins []string
	}{
		{
			name: "PR with approver and commenter",
			mockPRs: []*github.PullRequest{
				{
					Number:  github.Ptr(123),
					Title:   github.Ptr("Test PR"),
					Draft:   github.Ptr(false),
					HTMLURL: github.Ptr("https://github.com/owner/repo/pull/123"),
					User: &github.User{
						Login: github.Ptr("author"),
						Name:  github.Ptr("PR Author"),
					},
				},
			},
			mockTimelineEvents: map[int][]*github.Timeline{
				123: {
					NewReview("approver1", "Approver One", "approved"),
					NewReview("commenter1", "Commenter One", "commented"),
					NewReview("dependabot", "", "approved", "Bot"),
				},
			},
			expectedPRCount:         1,
			expectedApproverLogins:  []string{"approver1"},
			expectedCommenterLogins: []string{"commenter1"},
		},
		{
			name: "draft PR should be filtered out",
			mockPRs: []*github.PullRequest{
				{
					Number:  github.Ptr(124),
					Title:   github.Ptr("Draft PR"),
					Draft:   github.Ptr(true),
					HTMLURL: github.Ptr("https://github.com/owner/repo/pull/124"),
					User: &github.User{
						Login: github.Ptr("author"),
						Name:  github.Ptr("PR Author"),
					},
				},
			},
			mockTimelineEvents:      map[int][]*github.Timeline{},
			expectedPRCount:         0,
			expectedApproverLogins:  []string{},
			expectedCommenterLogins: []string{},
		},
		{
			name: "PR with no reviews",
			mockPRs: []*github.PullRequest{
				{
					Number:  github.Ptr(125),
					Title:   github.Ptr("No Reviews PR"),
					Draft:   github.Ptr(false),
					HTMLURL: github.Ptr("https://github.com/owner/repo/pull/125"),
					User: &github.User{
						Login: github.Ptr("author"),
						Name:  github.Ptr("PR Author"),
					},
				},
			},
			mockTimelineEvents:      map[int][]*github.Timeline{},
			expectedPRCount:         1,
			expectedApproverLogins:  []string{},
			expectedCommenterLogins: []string{},
		},
		{
			name: "approver who also commented - should only appear in approvers",
			mockPRs: []*github.PullRequest{
				{
					Number:  github.Ptr(126),
					Title:   github.Ptr("Approver Also Comments PR"),
					Draft:   github.Ptr(false),
					HTMLURL: github.Ptr("https://github.com/owner/repo/pull/126"),
					User: &github.User{
						Login: github.Ptr("author"),
						Name:  github.Ptr("PR Author"),
					},
				},
			},
			mockTimelineEvents: map[int][]*github.Timeline{
				126: {
					NewReview("reviewer1", "Reviewer One", "commented"),
					NewReview("reviewer1", "Reviewer One", "approved"),
				},
			},
			expectedPRCount:         1,
			expectedApproverLogins:  []string{"reviewer1"},
			expectedCommenterLogins: []string{},
		},
		{
			name: "author commenting own PR should be excluded from commenters",
			mockPRs: []*github.PullRequest{
				{
					Number:  github.Ptr(127),
					Title:   github.Ptr("Author Comments PR"),
					Draft:   github.Ptr(false),
					HTMLURL: github.Ptr("https://github.com/owner/repo/pull/127"),
					User: &github.User{
						Login: github.Ptr("pr-author"),
						Name:  github.Ptr("PR Author"),
					},
				},
			},
			mockTimelineEvents: map[int][]*github.Timeline{
				127: {
					NewReview("pr-author", "PR Author", "commented"),
					NewReview("external-reviewer", "External Reviewer", "approved"),
				},
			},
			expectedPRCount:         1,
			expectedApproverLogins:  []string{"external-reviewer"},
			expectedCommenterLogins: []string{},
		},
		{
			name: "bot reviews should be excluded completely",
			mockPRs: []*github.PullRequest{
				{
					Number:  github.Ptr(128),
					Title:   github.Ptr("Bot Reviews PR"),
					Draft:   github.Ptr(false),
					HTMLURL: github.Ptr("https://github.com/owner/repo/pull/128"),
					User: &github.User{
						Login: github.Ptr("author"),
						Name:  github.Ptr("PR Author"),
					},
				},
			},
			mockTimelineEvents: map[int][]*github.Timeline{
				128: {
					NewReview("dependabot[bot]", "", "approved", "Bot"),
					NewReview("codecov[bot]", "", "commented", "Bot"),
					NewReview("human-reviewer", "Human Reviewer", "commented"),
				},
			},
			expectedPRCount:         1,
			expectedApproverLogins:  []string{},
			expectedCommenterLogins: []string{"human-reviewer"},
		},
		{
			name: "invalid reviews should be excluded",
			mockPRs: []*github.PullRequest{
				{
					Number:  github.Ptr(129),
					Title:   github.Ptr("Invalid Reviews PR"),
					Draft:   github.Ptr(false),
					HTMLURL: github.Ptr("https://github.com/owner/repo/pull/129"),
					User: &github.User{
						Login: github.Ptr("author"),
						Name:  github.Ptr("PR Author"),
					},
				},
			},
			mockTimelineEvents: map[int][]*github.Timeline{
				129: {
					{ // nil user event retained for invalid case
						User:  nil,
						State: github.Ptr("approved"),
						Event: github.Ptr("reviewed"),
					},
					NewReview("", "Empty Login User", "commented"),
					NewReview("valid-reviewer", "Valid Reviewer", "approved"),
				},
			},
			expectedPRCount:         1,
			expectedApproverLogins:  []string{"valid-reviewer"},
			expectedCommenterLogins: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPRService := &mockPullRequestsService{
				mockPRs: tt.mockPRs,
				mockResponse: &github.Response{
					Response: &http.Response{
						StatusCode: 200,
					},
				},
				mockError: nil,
			}
			mockIssueService := &mockIssueService{
				mockTimelineEventsByPRNumber: tt.mockTimelineEvents,
				mockResponse: &github.Response{
					Response: &http.Response{
						StatusCode: 200,
					},
				},
				mockError: nil,
			}
			client := githubclient.NewClient(mockPRService, mockIssueService)

			repos := []models.Repository{
				{Owner: "testowner", Name: "testrepo"},
			}

			getFilters := func(repo models.Repository) config.Filters {
				return config.Filters{} // empty filters = allow all
			}

			result, err := client.FetchOpenPRs(repos, getFilters)

			if err != nil {
				t.Fatalf("FetchOpenPRs() returned error: %v", err)
			}

			if len(result) != tt.expectedPRCount {
				t.Errorf("Expected %d PRs, got %d", tt.expectedPRCount, len(result))
				return
			}

			if tt.expectedPRCount > 0 {
				pr := result[0]

				if pr.GetNumber() != *tt.mockPRs[0].Number {
					t.Errorf("Expected PR number %d, got %d", *tt.mockPRs[0].Number, pr.GetNumber())
				}

				actualApproverLogins := make([]string, len(pr.ApprovedByUsers))
				for i, user := range pr.ApprovedByUsers {
					actualApproverLogins[i] = user.Login
				}

				actualCommenterLogins := make([]string, len(pr.CommentedByUsers))
				for i, user := range pr.CommentedByUsers {
					actualCommenterLogins[i] = user.Login
				}

				if !slicesEqualIgnoreOrder(tt.expectedApproverLogins, actualApproverLogins) {
					t.Errorf("Expected approver logins %v, got %v", tt.expectedApproverLogins, actualApproverLogins)
				}

				if !slicesEqualIgnoreOrder(tt.expectedCommenterLogins, actualCommenterLogins) {
					t.Errorf("Expected commenter logins %v, got %v", tt.expectedCommenterLogins, actualCommenterLogins)
				}

				expectedRepo := models.Repository{Owner: "testowner", Name: "testrepo"}
				if pr.Repository != expectedRepo {
					t.Errorf("Expected repository %v, got %v", expectedRepo, pr.Repository)
				}

				expectedAuthor := *tt.mockPRs[0].User.Login
				if pr.Author.Login != expectedAuthor {
					t.Errorf("Expected author login %s, got %s", expectedAuthor, pr.Author.Login)
				}
			}
		})
	}
}

func TestFetchOpenPRs_MultipleRepositories(t *testing.T) {
	mockPRService1 := &mockPullRequestsService{mockPRs: []*github.PullRequest{{Number: github.Ptr(1), Title: github.Ptr("Repo1 PR"), Draft: github.Ptr(false), HTMLURL: github.Ptr("https://example.com/r1/1"), User: &github.User{Login: github.Ptr("author1")}}}, mockResponse: &github.Response{Response: &http.Response{StatusCode: 200}}, mockError: nil}
	mockPRService2 := &mockPullRequestsService{mockPRs: []*github.PullRequest{{Number: github.Ptr(2), Title: github.Ptr("Repo2 PR"), Draft: github.Ptr(false), HTMLURL: github.Ptr("https://example.com/r2/2"), User: &github.User{Login: github.Ptr("author2")}}}, mockResponse: &github.Response{Response: &http.Response{StatusCode: 200}}, mockError: nil}
	mockIssueService := &mockIssueService{mockTimelineEventsByPRNumber: map[int][]*github.Timeline{}, mockResponse: &github.Response{Response: &http.Response{StatusCode: 200}}, mockError: nil}

	client := githubclient.NewClient(&multiRepoPRService{services: map[string]*mockPullRequestsService{"repo1": mockPRService1, "repo2": mockPRService2}}, mockIssueService)
	repos := []models.Repository{{Owner: "o", Name: "repo1"}, {Owner: "o", Name: "repo2"}}
	result, err := client.FetchOpenPRs(repos, func(models.Repository) config.Filters { return config.Filters{} })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(result))
	}
	numbers := []int{result[0].GetNumber(), result[1].GetNumber()}
	if !((numbers[0] == 1 && numbers[1] == 2) || (numbers[0] == 2 && numbers[1] == 1)) {
		t.Errorf("expected PR numbers 1 and 2, got %v", numbers)
	}
}

func TestFetchOpenPRs_ErrorShortCircuits(t *testing.T) {
	mockPRService404 := &mockPullRequestsService{mockPRs: nil, mockResponse: &github.Response{Response: &http.Response{StatusCode: 404}}, mockError: fmt.Errorf("not found")}
	mockPRServiceOK := &mockPullRequestsService{mockPRs: []*github.PullRequest{{Number: github.Ptr(3), Title: github.Ptr("Ok PR"), Draft: github.Ptr(false), HTMLURL: github.Ptr("https://example.com/r/3"), User: &github.User{Login: github.Ptr("author")}}}, mockResponse: &github.Response{Response: &http.Response{StatusCode: 200}}, mockError: nil}
	mockIssueService := &mockIssueService{mockTimelineEventsByPRNumber: map[int][]*github.Timeline{}, mockResponse: &github.Response{Response: &http.Response{StatusCode: 200}}, mockError: nil}

	client := githubclient.NewClient(&multiRepoPRService{services: map[string]*mockPullRequestsService{"bad": mockPRService404, "good": mockPRServiceOK}}, mockIssueService)
	repos := []models.Repository{{Owner: "o", Name: "bad"}, {Owner: "o", Name: "good"}}
	_, err := client.FetchOpenPRs(repos, func(models.Repository) config.Filters { return config.Filters{} })
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestFetchOpenPRs_ConcurrencyLimit(t *testing.T) {
	repoCount := githubclient.DefaultRepoFetchConcurrencyLimit + 3
	services := make(map[string]*mockPullRequestsService)
	repos := make([]models.Repository, 0, repoCount)
	for i := 0; i < repoCount; i++ {
		name := fmt.Sprintf("repo-%d", i)
		services[name] = &mockPullRequestsService{mockPRs: []*github.PullRequest{{Number: github.Ptr(i + 100), Title: github.Ptr(name + " PR"), Draft: github.Ptr(false), HTMLURL: github.Ptr("https://example.com/" + name), User: &github.User{Login: github.Ptr("author")}}}, mockResponse: &github.Response{Response: &http.Response{StatusCode: 200}}, mockError: nil}
		repos = append(repos, models.Repository{Owner: "o", Name: name})
	}
	mockIssueService := &mockIssueService{mockTimelineEventsByPRNumber: map[int][]*github.Timeline{}, mockResponse: &github.Response{Response: &http.Response{StatusCode: 200}}, mockError: nil}
	client := githubclient.NewClient(&multiRepoPRService{services: services}, mockIssueService)
	prs, err := client.FetchOpenPRs(repos, func(models.Repository) config.Filters { return config.Filters{} })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != repoCount {
		t.Fatalf("expected %d PRs, got %d", repoCount, len(prs))
	}
}

func slicesEqualIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	mapA := make(map[string]bool)
	for _, v := range a {
		mapA[v] = true
	}
	for _, v := range b {
		if !mapA[v] {
			return false
		}
	}
	return true
}
