// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/organization"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	notify_service "forgejo.org/services/notify"
)

// DeleteNotPassedAssignee deletes all assignees who aren't passed via the "assignees" array
func DeleteNotPassedAssignee(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, assignees []*user_model.User) (err error) {
	var found bool
	oriAssignes := make([]*user_model.User, len(issue.Assignees))
	_ = copy(oriAssignes, issue.Assignees)

	for _, assignee := range oriAssignes {
		found = false
		for _, alreadyAssignee := range assignees {
			if assignee.ID == alreadyAssignee.ID {
				found = true
				break
			}
		}

		if !found {
			// This function also does comments and hooks, which is why we call it separately instead of directly removing the assignees here
			if _, _, err := ToggleAssigneeWithNotify(ctx, issue, doer, assignee.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// ToggleAssigneeWithNoNotify changes a user between assigned and not assigned for this issue, and make issue comment for it.
func ToggleAssigneeWithNotify(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, assigneeID int64) (removed bool, comment *issues_model.Comment, err error) {
	removed, comment, err = issues_model.ToggleIssueAssignee(ctx, issue, doer, assigneeID)
	if err != nil {
		return false, nil, err
	}

	assignee, err := user_model.GetUserByID(ctx, assigneeID)
	if err != nil {
		return false, nil, err
	}

	notify_service.IssueChangeAssignee(ctx, doer, issue, assignee, removed, comment)

	return removed, comment, err
}

// ReviewRequest add or remove a review request from a user for this PR, and make comment for it.
func ReviewRequest(ctx context.Context, issue *issues_model.Issue, doer, reviewer *user_model.User, isAdd bool) (comment *issues_model.Comment, err error) {
	if isAdd {
		comment, err = issues_model.AddReviewRequest(ctx, issue, reviewer, doer)
	} else {
		comment, err = issues_model.RemoveReviewRequest(ctx, issue, reviewer, doer)
	}

	if err != nil {
		return nil, err
	}

	// don't notify if the user is requesting itself as reviewer
	if comment != nil && doer.ID != reviewer.ID {
		notify_service.PullRequestReviewRequest(ctx, doer, issue, reviewer, isAdd, comment)
	}

	return comment, err
}

// IsValidReviewRequest Check permission for ReviewRequest
func IsValidReviewRequest(ctx context.Context, reviewer, doer *user_model.User, isAdd bool, issue *issues_model.Issue, permDoer *access_model.Permission) error {
	if reviewer.IsOrganization() {
		return issues_model.ErrNotValidReviewRequest{
			Reason: "Organization can't be added as reviewer",
			UserID: doer.ID,
			RepoID: issue.Repo.ID,
		}
	}
	if doer.IsOrganization() {
		return issues_model.ErrNotValidReviewRequest{
			Reason: "Organization can't be doer to add reviewer",
			UserID: doer.ID,
			RepoID: issue.Repo.ID,
		}
	}

	permReviewer, err := access_model.GetUserRepoPermission(ctx, issue.Repo, reviewer)
	if err != nil {
		return err
	}

	if permDoer == nil {
		permDoer = new(access_model.Permission)
		*permDoer, err = access_model.GetUserRepoPermission(ctx, issue.Repo, doer)
		if err != nil {
			return err
		}
	}

	lastreview, err := issues_model.GetReviewByIssueIDAndUserID(ctx, issue.ID, reviewer.ID)
	if err != nil && !issues_model.IsErrReviewNotExist(err) {
		return err
	}

	canDoerChangeReviewRequests := CanDoerChangeReviewRequests(ctx, doer, issue.Repo, issue)

	if isAdd {
		if !permReviewer.CanAccessAny(perm.AccessModeRead, unit.TypePullRequests) {
			return issues_model.ErrNotValidReviewRequest{
				Reason: "Reviewer can't read",
				UserID: doer.ID,
				RepoID: issue.Repo.ID,
			}
		}

		if reviewer.ID == issue.PosterID && issue.OriginalAuthorID == 0 {
			return issues_model.ErrNotValidReviewRequest{
				Reason: "poster of pr can't be reviewer",
				UserID: doer.ID,
				RepoID: issue.Repo.ID,
			}
		}

		if canDoerChangeReviewRequests {
			return nil
		}

		if doer.ID == issue.PosterID && issue.OriginalAuthorID == 0 && lastreview != nil && lastreview.Type != issues_model.ReviewTypeRequest {
			return nil
		}

		return issues_model.ErrNotValidReviewRequest{
			Reason: "Doer can't choose reviewer",
			UserID: doer.ID,
			RepoID: issue.Repo.ID,
		}
	}

	if canDoerChangeReviewRequests {
		return nil
	}

	if lastreview != nil && lastreview.Type == issues_model.ReviewTypeRequest && lastreview.ReviewerID == doer.ID {
		return nil
	}

	return issues_model.ErrNotValidReviewRequest{
		Reason: "Doer can't remove reviewer",
		UserID: doer.ID,
		RepoID: issue.Repo.ID,
	}
}

// IsValidTeamReviewRequest Check permission for ReviewRequest Team
func IsValidTeamReviewRequest(ctx context.Context, reviewer *organization.Team, doer *user_model.User, isAdd bool, issue *issues_model.Issue) error {
	if doer.IsOrganization() {
		return issues_model.ErrNotValidReviewRequest{
			Reason: "Organization can't be doer to add reviewer",
			UserID: doer.ID,
			RepoID: issue.Repo.ID,
		}
	}

	canDoerChangeReviewRequests := CanDoerChangeReviewRequests(ctx, doer, issue.Repo, issue)

	if isAdd {
		if issue.Repo.IsPrivate {
			hasTeam := organization.HasTeamRepo(ctx, reviewer.OrgID, reviewer.ID, issue.RepoID)

			if !hasTeam {
				return issues_model.ErrNotValidReviewRequest{
					Reason: "Reviewing team can't read repo",
					UserID: doer.ID,
					RepoID: issue.Repo.ID,
				}
			}
		}

		if canDoerChangeReviewRequests {
			return nil
		}

		return issues_model.ErrNotValidReviewRequest{
			Reason: "Doer can't choose reviewer",
			UserID: doer.ID,
			RepoID: issue.Repo.ID,
		}
	}

	if canDoerChangeReviewRequests {
		return nil
	}

	return issues_model.ErrNotValidReviewRequest{
		Reason: "Doer can't remove reviewer",
		UserID: doer.ID,
		RepoID: issue.Repo.ID,
	}
}

// TeamReviewRequest add or remove a review request from a team for this PR, and make comment for it.
func TeamReviewRequest(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, reviewer *organization.Team, isAdd bool) (comment *issues_model.Comment, err error) {
	if isAdd {
		comment, err = issues_model.AddTeamReviewRequest(ctx, issue, reviewer, doer)
	} else {
		comment, err = issues_model.RemoveTeamReviewRequest(ctx, issue, reviewer, doer)
	}

	if err != nil {
		return nil, err
	}

	if comment == nil || !isAdd {
		return nil, nil
	}

	return comment, teamReviewRequestNotify(ctx, issue, doer, reviewer, isAdd, comment)
}

func ReviewRequestNotify(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, reviewNotifers []*ReviewRequestNotifier) {
	for _, reviewNotifer := range reviewNotifers {
		if reviewNotifer.Reviewer != nil {
			notify_service.PullRequestReviewRequest(ctx, issue.Poster, issue, reviewNotifer.Reviewer, reviewNotifer.IsAdd, reviewNotifer.Comment)
		} else if reviewNotifer.ReviewTeam != nil {
			if err := teamReviewRequestNotify(ctx, issue, issue.Poster, reviewNotifer.ReviewTeam, reviewNotifer.IsAdd, reviewNotifer.Comment); err != nil {
				log.Error("teamReviewRequestNotify: %v", err)
			}
		}
	}
}

// teamReviewRequestNotify notify all user in this team
func teamReviewRequestNotify(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, reviewer *organization.Team, isAdd bool, comment *issues_model.Comment) error {
	// notify all user in this team
	if err := comment.LoadIssue(ctx); err != nil {
		return err
	}

	members, err := organization.GetTeamMembers(ctx, &organization.SearchMembersOptions{
		TeamID: reviewer.ID,
	})
	if err != nil {
		return err
	}

	for _, member := range members {
		if member.ID == comment.Issue.PosterID {
			continue
		}
		comment.AssigneeID = member.ID
		notify_service.PullRequestReviewRequest(ctx, doer, issue, member, isAdd, comment)
	}

	return err
}

// CanDoerChangeReviewRequests returns if the doer can add/remove review requests of a PR
func CanDoerChangeReviewRequests(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, issue *issues_model.Issue) bool {
	// The poster of the PR can change the reviewers
	if doer.ID == issue.PosterID {
		return true
	}

	// The owner of the repo can change the reviewers
	if doer.ID == repo.OwnerID {
		return true
	}

	// Collaborators of the repo can change the reviewers
	isCollaborator, err := repo_model.IsCollaborator(ctx, repo.ID, doer.ID)
	if err != nil {
		log.Error("IsCollaborator: %v", err)
		return false
	}
	if isCollaborator {
		return true
	}

	// If the repo's owner is an organization, members of teams with read permission on pull requests can change reviewers
	if repo.Owner.IsOrganization() {
		teams, err := organization.GetTeamsWithAccessToRepo(ctx, repo.OwnerID, repo.ID, perm.AccessModeRead)
		if err != nil {
			log.Error("GetTeamsWithAccessToRepo: %v", err)
			return false
		}
		for _, team := range teams {
			if !team.UnitEnabled(ctx, unit.TypePullRequests) {
				continue
			}
			isMember, err := organization.IsTeamMember(ctx, repo.OwnerID, team.ID, doer.ID)
			if err != nil {
				log.Error("IsTeamMember: %v", err)
				continue
			}
			if isMember {
				return true
			}
		}
	}

	return false
}
