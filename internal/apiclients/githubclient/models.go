package githubclient

import (
	"cmp"
	"log"
	"slices"

	"github.com/google/go-github/v72/github"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
)

type PR struct {
	*github.PullRequest
	Repository       models.Repository
	Author           Collaborator
	ApprovedByUsers  []Collaborator
	CommentedByUsers []Collaborator // reviewers who commented the PR but did not approve it
}

type PRResult struct {
	pr         *github.PullRequest
	repository models.Repository
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
	eventsWithValidUser := utilities.Filter(r.timelineEvents, hasValidReviewerUserData)

	approvingReviews := utilities.Filter(eventsWithValidUser, isApprovingReview)
	approvedByUsers := extractUniqueCollaboratorsFromReviews(approvingReviews)

	allReviewsAndComments := utilities.Filter(eventsWithValidUser, isReviewEvent)
	commentedByUsers := utilities.Filter(
		extractUniqueCollaboratorsFromReviews(allReviewsAndComments),
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

func hasValidReviewerUserData(r *github.Timeline) bool {
	user := r.GetUser()
	return user != nil && user.GetLogin() != "" && !isBot(user)
}

func extractUniqueCollaboratorsFromReviews(events []*github.Timeline) []Collaborator {
	return utilities.UniqueFunc(
		utilities.Map(events, getCollaborator), isUniqueCollaborator,
	)
}

func getCollaborator(r *github.Timeline) Collaborator {
	return newCollaboratorFromUser(r.GetUser())
}

func isUniqueCollaborator(a, b Collaborator) bool {
	return a.Login == b.Login
}

func isApprovingReview(event *github.Timeline) bool {
	return isReviewEvent(event) && event.GetState() == "approved"

}

// includes diff & timeline comments too
func isReviewEvent(event *github.Timeline) bool {
	return event.GetEvent() == "reviewed"
}

func getFilterForCommenters(authorLogin string, approvedByUsers []Collaborator) func(c Collaborator) bool {
	return func(c Collaborator) bool {
		return c.Login != authorLogin &&
			!slices.ContainsFunc(approvedByUsers, func(approver Collaborator) bool {
				return c.Login == approver.Login
			})
	}
}
