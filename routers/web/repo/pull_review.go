// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"

	issues_model "forgejo.org/models/issues"
	pull_model "forgejo.org/models/pull"
	"forgejo.org/modules/base"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/context/upload"
	"forgejo.org/services/forms"
	pull_service "forgejo.org/services/pull"
)

const (
	tplDiffConversation     base.TplName = "repo/diff/conversation"
	tplTimelineConversation base.TplName = "repo/issue/view_content/conversation"
	tplNewComment           base.TplName = "repo/diff/new_comment"
)

// RenderNewCodeCommentForm will render the form for creating a new review comment
func RenderNewCodeCommentForm(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	if ctx.Written() {
		return
	}
	if !issue.IsPull {
		return
	}
	currentReview, err := issues_model.GetCurrentReview(ctx, ctx.Doer, issue)
	if err != nil && !issues_model.IsErrReviewNotExist(err) {
		ctx.ServerError("GetCurrentReview", err)
		return
	}
	ctx.Data["PageIsPullFiles"] = true
	ctx.Data["Issue"] = issue
	ctx.Data["CurrentReview"] = currentReview
	pullHeadCommitID, err := ctx.Repo.GitRepo.GetRefCommitID(issue.PullRequest.GetGitRefName())
	if err != nil {
		ctx.ServerError("GetRefCommitID", err)
		return
	}
	ctx.Data["AfterCommitID"] = pullHeadCommitID
	ctx.Data["IsAttachmentEnabled"] = setting.Attachment.Enabled
	upload.AddUploadContext(ctx, "comment")
	ctx.HTML(http.StatusOK, tplNewComment)
}

// CreateCodeComment will create a code comment including an pending review if required
func CreateCodeComment(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CodeCommentForm)
	issue := GetActionIssue(ctx)
	if ctx.Written() {
		return
	}
	if !issue.IsPull {
		return
	}

	if ctx.HasError() {
		ctx.Flash.Error(ctx.Data["ErrorMsg"].(string))
		ctx.Redirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		return
	}

	signedLine := form.Line
	if form.Side == "previous" {
		signedLine *= -1
	}

	var attachments []string
	if setting.Attachment.Enabled {
		attachments = form.Files
	}

	// If the reply is made to a comment that is part of a pending review, then
	// this comment also should be seen as part of that pending review. Consider
	// it to be a pending review by default, except when `single_review` was
	// passed.
	pendingReview := !form.SingleReview
	if form.Reply > 0 {
		r, err := issues_model.GetReviewByID(ctx, form.Reply)
		if err != nil {
			ctx.ServerError("GetReviewByID", err)
			return
		}
		if r.IssueID != issue.ID {
			ctx.NotFound("Review does not belong to pull request", nil)
			return
		}
		pendingReview = r.Type == issues_model.ReviewTypePending
	}

	comment, err := pull_service.CreateCodeComment(ctx,
		ctx.Doer,
		ctx.Repo.GitRepo,
		issue,
		signedLine,
		form.Content,
		form.TreePath,
		pendingReview,
		form.Reply,
		form.LatestCommitID,
		attachments,
	)
	if err != nil {
		ctx.ServerError("CreateCodeComment", err)
		return
	}

	if comment == nil {
		log.Trace("Comment not created: %-v #%d[%d]", ctx.Repo.Repository, issue.Index, issue.ID)
		ctx.Redirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		return
	}

	log.Trace("Comment created: %-v #%d[%d] Comment[%d]", ctx.Repo.Repository, issue.Index, issue.ID, comment.ID)

	renderConversation(ctx, comment, form.Origin)
}

// UpdateResolveConversation add or remove an Conversation resolved mark
func UpdateResolveConversation(ctx *context.Context) {
	origin := ctx.FormString("origin")
	action := ctx.FormString("action")
	commentID := ctx.FormInt64("comment_id")

	comment, err := issues_model.GetCommentByID(ctx, commentID)
	if err != nil {
		ctx.ServerError("GetIssueByID", err)
		return
	}

	if err = comment.LoadIssue(ctx); err != nil {
		ctx.ServerError("comment.LoadIssue", err)
		return
	}

	if comment.Issue.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("comment's repoID is incorrect", errors.New("comment's repoID is incorrect"))
		return
	}

	var permResult bool
	if permResult, err = issues_model.CanMarkConversation(ctx, comment.Issue, ctx.Doer); err != nil {
		ctx.ServerError("CanMarkConversation", err)
		return
	}
	if !permResult {
		ctx.Error(http.StatusForbidden)
		return
	}

	if !comment.Issue.IsPull {
		ctx.Error(http.StatusBadRequest)
		return
	}

	if action == "Resolve" || action == "UnResolve" {
		err = issues_model.MarkConversation(ctx, comment, ctx.Doer, action == "Resolve")
		if err != nil {
			ctx.ServerError("MarkConversation", err)
			return
		}
	} else {
		ctx.Error(http.StatusBadRequest)
		return
	}

	renderConversation(ctx, comment, origin)
}

func renderConversation(ctx *context.Context, comment *issues_model.Comment, origin string) {
	comments, err := issues_model.FetchCodeConversation(ctx, comment, ctx.Doer)
	if err != nil {
		ctx.ServerError("FetchCodeCommentsByLine", err)
		return
	}
	ctx.Data["PageIsPullFiles"] = (origin == "diff")

	if err := comments.LoadAttachments(ctx); err != nil {
		ctx.ServerError("LoadAttachments", err)
		return
	}

	ctx.Data["IsAttachmentEnabled"] = setting.Attachment.Enabled
	upload.AddUploadContext(ctx, "comment")

	ctx.Data["comments"] = comments
	if ctx.Data["CanMarkConversation"], err = issues_model.CanMarkConversation(ctx, comment.Issue, ctx.Doer); err != nil {
		ctx.ServerError("CanMarkConversation", err)
		return
	}
	ctx.Data["Issue"] = comment.Issue
	if err = comment.Issue.LoadPullRequest(ctx); err != nil {
		ctx.ServerError("comment.Issue.LoadPullRequest", err)
		return
	}
	pullHeadCommitID, err := ctx.Repo.GitRepo.GetRefCommitID(comment.Issue.PullRequest.GetGitRefName())
	if err != nil {
		ctx.ServerError("GetRefCommitID", err)
		return
	}
	ctx.Data["AfterCommitID"] = pullHeadCommitID
	if origin == "diff" {
		ctx.HTML(http.StatusOK, tplDiffConversation)
	} else if origin == "timeline" {
		ctx.HTML(http.StatusOK, tplTimelineConversation)
	}
}

// SubmitReview creates a review out of the existing pending review or creates a new one if no pending review exist
func SubmitReview(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.SubmitReviewForm)
	issue := GetActionIssue(ctx)
	if ctx.Written() {
		return
	}
	if !issue.IsPull {
		return
	}
	if ctx.HasError() {
		ctx.Flash.Error(ctx.Data["ErrorMsg"].(string))
		ctx.JSONRedirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		return
	}

	reviewType := form.ReviewType()
	switch reviewType {
	case issues_model.ReviewTypeUnknown:
		ctx.ServerError("ReviewType", fmt.Errorf("unknown ReviewType: %s", form.Type))
		return

	// can not approve/reject your own PR
	case issues_model.ReviewTypeApprove, issues_model.ReviewTypeReject:
		if issue.IsPoster(ctx.Doer.ID) {
			var translated string
			if reviewType == issues_model.ReviewTypeApprove {
				translated = ctx.Locale.TrString("repo.issues.review.self.approval")
			} else {
				translated = ctx.Locale.TrString("repo.issues.review.self.rejection")
			}

			ctx.Flash.Error(translated)
			ctx.JSONRedirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
			return
		}
	}

	var attachments []string
	if setting.Attachment.Enabled {
		attachments = form.Files
	}

	_, comm, err := pull_service.SubmitReview(ctx, ctx.Doer, ctx.Repo.GitRepo, issue, reviewType, form.Content, form.CommitID, attachments)
	if err != nil {
		if issues_model.IsContentEmptyErr(err) {
			ctx.Flash.Error(ctx.Tr("repo.issues.review.content.empty"))
			ctx.JSONRedirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		} else {
			ctx.ServerError("SubmitReview", err)
		}
		return
	}
	ctx.JSONRedirect(fmt.Sprintf("%s/pulls/%d#%s", ctx.Repo.RepoLink, issue.Index, comm.HashTag()))
}

// DismissReview dismissing stale review by repo admin
func DismissReview(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.DismissReviewForm)
	comm, err := pull_service.DismissReview(ctx, form.ReviewID, ctx.Repo.Repository.ID, form.Message, ctx.Doer, true, true)
	if err != nil {
		if pull_service.IsErrDismissRequestOnClosedPR(err) {
			ctx.Status(http.StatusForbidden)
			return
		}
		ctx.ServerError("pull_service.DismissReview", err)
		return
	}

	ctx.Redirect(fmt.Sprintf("%s/pulls/%d#%s", ctx.Repo.RepoLink, comm.Issue.Index, comm.HashTag()))
}

// viewedFilesUpdate Struct to parse the body of a request to update the reviewed files of a PR
// If you want to implement an API to update the review, simply move this struct into modules.
type viewedFilesUpdate struct {
	Files         map[string]bool `json:"files"`
	HeadCommitSHA string          `json:"headCommitSHA"`
}

func UpdateViewedFiles(ctx *context.Context) {
	// Find corresponding PR
	issue, ok := getPullInfo(ctx)
	if !ok {
		return
	}
	pull := issue.PullRequest

	var data *viewedFilesUpdate
	err := json.NewDecoder(ctx.Req.Body).Decode(&data)
	if err != nil {
		log.Warn("Attempted to update a review but could not parse request body: %v", err)
		ctx.Resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// Expect the review to have been now if no head commit was supplied
	if data.HeadCommitSHA == "" {
		data.HeadCommitSHA = pull.HeadCommitID
	}

	updatedFiles := make(map[string]pull_model.ViewedState, len(data.Files))
	for file, viewed := range data.Files {
		// Only unviewed and viewed are possible, has-changed can not be set from the outside
		state := pull_model.Unviewed
		if viewed {
			state = pull_model.Viewed
		}
		updatedFiles[file] = state
	}

	if err := pull_model.UpdateReviewState(ctx, ctx.Doer.ID, pull.ID, data.HeadCommitSHA, updatedFiles); err != nil {
		ctx.ServerError("UpdateReview", err)
	}
}
