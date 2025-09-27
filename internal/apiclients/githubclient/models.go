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

type FetchReviewsResult struct {
	pr         *github.PullRequest
	reviews    []*github.PullRequestReview
	repository models.Repository
	err        error
}

func (r FetchReviewsResult) printResult() {
	if r.err != nil {
		log.Printf("Unable to fetch reviews for PR #%d: %v", r.pr.GetNumber(), r.err)
	} else {
		log.Printf("Found %d reviews for PR %v/%d", len(r.reviews), r.repository, r.pr.GetNumber())
	}
	for _, review := range r.reviews {
		log.Printf(
			"Review by %s (name: %s): %s",
			review.GetUser().GetLogin(),
			review.GetUser().GetName(),
			*review.State,
		)
	}
}

type Collaborator struct {
	Login string // GitHub username
	Name  string // GitHub name if available
}

func NewCollaboratorFromUser(user *github.User) Collaborator {
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

func (r FetchReviewsResult) asPR() PR {
	authorLogin := r.pr.GetUser().GetLogin()
	reviewsWithValidUser := utilities.Filter(r.reviews, hasValidReviewerUserData)

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
		Author:           NewCollaboratorFromUser(r.pr.GetUser()),
		ApprovedByUsers:  approvedByUsers,
		CommentedByUsers: commentedByUsers,
	}
}

func hasValidReviewerUserData(r *github.PullRequestReview) bool {
	user := r.GetUser()
	return user != nil && user.GetLogin() != "" && !isBot(user)
}

func extractUniqueCollaboratorsFromReviews(reviews []*github.PullRequestReview) []Collaborator {
	return utilities.UniqueFunc(
		utilities.Map(reviews, getCollaborator), isUniqueCollaborator,
	)
}

func getCollaborator(r *github.PullRequestReview) Collaborator {
	return NewCollaboratorFromUser(r.GetUser())
}

func isUniqueCollaborator(a, b Collaborator) bool {
	return a.Login == b.Login
}

func getIsApprovedFilter(requiredApprovalStatus bool) func(review *github.PullRequestReview) bool {
	return func(review *github.PullRequestReview) bool {
		isApproved := review.GetState() == "APPROVED"
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
