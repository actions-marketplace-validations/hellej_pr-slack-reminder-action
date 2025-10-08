package githubclient

import (
	"cmp"
	"log"
	"slices"

	"github.com/google/go-github/v72/github"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
)

type PRsOfRepoResult struct {
	prs        []*github.PullRequest
	repository models.Repository
	err        error
}

func (r PRsOfRepoResult) GetPRCount() int {
	if r.prs != nil {
		return len(r.prs)
	}
	return 0
}

type PR struct {
	*github.PullRequest
	Repository       models.Repository
	Author           Collaborator
	ApprovedByUsers  []Collaborator
	CommentedByUsers []Collaborator // reviewers who commented the PR but did not approve it
}

type FetchTimelineResult struct {
	pr             *github.PullRequest
	timelineEvents []*github.Timeline
	repository     models.Repository
	err            error
}

func (r FetchTimelineResult) printResult() {
	if r.err != nil {
		log.Printf("Unable to fetch timeline events for PR #%d: %v", r.pr.GetNumber(), r.err)
	} else {
		log.Printf("Found %d timeline events for PR %v/%d", len(r.timelineEvents), r.repository, r.pr.GetNumber())
	}
	for _, event := range r.timelineEvents {
		log.Printf(
			"Event: %s, from: %s, state: %s",
			event.GetEvent(),
			event.GetUser().GetLogin(),
			event.GetState(),
		)
	}
}

type Collaborator struct {
	Login string // GitHub username
	Name  string // GitHub name if available
}

func newCollaboratorFromUser(user *github.User) Collaborator {
	return Collaborator{
		Login: user.GetLogin(),
		Name:  user.GetName(),
	}
}

func isBot(user *github.User) bool {
	userType := user.GetType()
	return userType == "Bot"
}

// Returns the GitHub name if available, otherwise login.
func (c Collaborator) GetGitHubName() string {
	return cmp.Or(c.Name, c.Login)
}

func (r FetchTimelineResult) asPR() PR {
	authorLogin := r.pr.GetUser().GetLogin()
	reviewsWithValidUser := utilities.Filter(
		utilities.Filter(r.timelineEvents, isReviewEvent),
		hasValidReviewerUserData,
	)
	approvingReviews := utilities.Filter(reviewsWithValidUser, getIsApprovedFilter(true))
	approvedByUsers := extractUniqueCollaboratorsFromReviews(approvingReviews)

	commentingReviews := utilities.Filter(reviewsWithValidUser, getIsApprovedFilter(false))
	commentedByUsers := utilities.Filter(
		extractUniqueCollaboratorsFromReviews(commentingReviews),
		getFilterForCommenters(authorLogin, approvedByUsers),
	)

	return PR{
		PullRequest:      r.pr,
		Repository:       r.repository,
		Author:           newCollaboratorFromUser(r.pr.GetUser()),
		ApprovedByUsers:  approvedByUsers,
		CommentedByUsers: commentedByUsers,
	}
}

func isReviewEvent(event *github.Timeline) bool {
	return event.GetEvent() == "reviewed"
}

func hasValidReviewerUserData(r *github.Timeline) bool {
	user := r.GetUser()
	return user != nil && user.GetLogin() != "" && !isBot(user)
}

func extractUniqueCollaboratorsFromReviews(reviews []*github.Timeline) []Collaborator {
	return utilities.UniqueFunc(
		utilities.Map(reviews, getCollaborator), isUniqueCollaborator,
	)
}

func getCollaborator(r *github.Timeline) Collaborator {
	return newCollaboratorFromUser(r.GetUser())
}

func isUniqueCollaborator(a, b Collaborator) bool {
	return a.Login == b.Login
}

func getIsApprovedFilter(requiredApprovalStatus bool) func(review *github.Timeline) bool {
	return func(review *github.Timeline) bool {
		isApproved := review.GetState() == "approved"
		return isApproved == requiredApprovalStatus
	}
}

func getFilterForCommenters(authorLogin string, approvedByUsers []Collaborator) func(c Collaborator) bool {
	return func(c Collaborator) bool {
		return c.Login != authorLogin &&
			!slices.ContainsFunc(approvedByUsers, func(approver Collaborator) bool {
				return c.Login == approver.Login
			})
	}
}
