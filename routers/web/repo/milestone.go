// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/modules/base"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
	"forgejo.org/services/issue"

	"xorm.io/builder"
)

const (
	tplMilestone       base.TplName = "repo/issue/milestones"
	tplMilestoneNew    base.TplName = "repo/issue/milestone_new"
	tplMilestoneIssues base.TplName = "repo/issue/milestone_issues"
)

// Milestones render milestones page
func Milestones(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.milestones")
	ctx.Data["PageIsIssueList"] = true
	ctx.Data["PageIsMilestones"] = true

	isShowClosed := ctx.FormString("state") == "closed"
	sortType := ctx.FormString("sort")
	keyword := ctx.FormTrim("q")
	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	miles, total, err := db.FindAndCount[issues_model.Milestone](ctx, issues_model.FindMilestoneOptions{
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: setting.UI.IssuePagingNum,
		},
		RepoID:   ctx.Repo.Repository.ID,
		IsClosed: optional.Some(isShowClosed),
		SortType: sortType,
		Name:     keyword,
	})
	if err != nil {
		ctx.ServerError("GetMilestones", err)
		return
	}

	stats, err := issues_model.GetMilestonesStatsByRepoCondAndKw(ctx, builder.And(builder.Eq{"id": ctx.Repo.Repository.ID}), keyword)
	if err != nil {
		ctx.ServerError("GetMilestoneStats", err)
		return
	}
	ctx.Data["OpenCount"] = stats.OpenCount
	ctx.Data["ClosedCount"] = stats.ClosedCount
	linkStr := "%s/milestones?state=%s&q=%s&sort=%s"
	ctx.Data["OpenLink"] = fmt.Sprintf(linkStr, ctx.Repo.RepoLink, "open",
		url.QueryEscape(keyword), url.QueryEscape(sortType))
	ctx.Data["ClosedLink"] = fmt.Sprintf(linkStr, ctx.Repo.RepoLink, "closed",
		url.QueryEscape(keyword), url.QueryEscape(sortType))

	if ctx.Repo.Repository.IsTimetrackerEnabled(ctx) {
		if err := issues_model.MilestoneList(miles).LoadTotalTrackedTimes(ctx); err != nil {
			ctx.ServerError("LoadTotalTrackedTimes", err)
			return
		}
	}
	for _, m := range miles {
		m.RenderedContent, err = markdown.RenderString(&markup.RenderContext{
			Links: markup.Links{
				Base: ctx.Repo.RepoLink,
			},
			Metas:   ctx.Repo.Repository.ComposeMetas(ctx),
			GitRepo: ctx.Repo.GitRepo,
			Ctx:     ctx,
		}, m.Content)
		if err != nil {
			ctx.ServerError("RenderString", err)
			return
		}
	}
	ctx.Data["Milestones"] = miles

	if isShowClosed {
		ctx.Data["State"] = "closed"
	} else {
		ctx.Data["State"] = "open"
	}

	ctx.Data["SortType"] = sortType
	ctx.Data["Keyword"] = keyword
	ctx.Data["IsShowClosed"] = isShowClosed

	pager := context.NewPagination(int(total), setting.UI.IssuePagingNum, page, 5)
	pager.AddParam(ctx, "state", "State")
	pager.AddParam(ctx, "q", "Keyword")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplMilestone)
}

// NewMilestone render creating milestone page
func NewMilestone(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.milestones.new")
	ctx.Data["PageIsIssueList"] = true
	ctx.Data["PageIsMilestones"] = true
	ctx.HTML(http.StatusOK, tplMilestoneNew)
}

// NewMilestonePost response for creating milestone
func NewMilestonePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateMilestoneForm)
	ctx.Data["Title"] = ctx.Tr("repo.milestones.new")
	ctx.Data["PageIsIssueList"] = true
	ctx.Data["PageIsMilestones"] = true

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplMilestoneNew)
		return
	}

	if len(form.Deadline) == 0 {
		form.Deadline = "9999-12-31"
	}
	deadline, err := time.ParseInLocation("2006-01-02", form.Deadline, time.Local)
	if err != nil {
		ctx.Data["Err_Deadline"] = true
		ctx.RenderWithErr(ctx.Tr("repo.milestones.invalid_due_date_format"), tplMilestoneNew, &form)
		return
	}

	deadline = time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 23, 59, 59, 0, deadline.Location())
	if err = issues_model.NewMilestone(ctx, &issues_model.Milestone{
		RepoID:       ctx.Repo.Repository.ID,
		Name:         form.Title,
		Content:      form.Content,
		DeadlineUnix: timeutil.TimeStamp(deadline.Unix()),
	}); err != nil {
		ctx.ServerError("NewMilestone", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.milestones.create_success", form.Title))
	ctx.Redirect(ctx.Repo.RepoLink + "/milestones")
}

// EditMilestone render edting milestone page
func EditMilestone(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.milestones.edit")
	ctx.Data["PageIsMilestones"] = true
	ctx.Data["PageIsEditMilestone"] = true

	m, err := issues_model.GetMilestoneByRepoID(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":id"))
	if err != nil {
		if issues_model.IsErrMilestoneNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetMilestoneByRepoID", err)
		}
		return
	}
	ctx.Data["title"] = m.Name
	ctx.Data["content"] = m.Content
	if len(m.DeadlineString) > 0 {
		ctx.Data["deadline"] = m.DeadlineString
	}
	ctx.HTML(http.StatusOK, tplMilestoneNew)
}

// EditMilestonePost response for edting milestone
func EditMilestonePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateMilestoneForm)
	ctx.Data["Title"] = ctx.Tr("repo.milestones.edit")
	ctx.Data["PageIsMilestones"] = true
	ctx.Data["PageIsEditMilestone"] = true

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplMilestoneNew)
		return
	}

	if len(form.Deadline) == 0 {
		form.Deadline = "9999-12-31"
	}
	deadline, err := time.ParseInLocation("2006-01-02", form.Deadline, time.Local)
	if err != nil {
		ctx.Data["Err_Deadline"] = true
		ctx.RenderWithErr(ctx.Tr("repo.milestones.invalid_due_date_format"), tplMilestoneNew, &form)
		return
	}

	deadline = time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 23, 59, 59, 0, deadline.Location())
	m, err := issues_model.GetMilestoneByRepoID(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":id"))
	if err != nil {
		if issues_model.IsErrMilestoneNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetMilestoneByRepoID", err)
		}
		return
	}
	m.Name = form.Title
	m.Content = form.Content
	m.DeadlineUnix = timeutil.TimeStamp(deadline.Unix())
	if err = issues_model.UpdateMilestone(ctx, m, m.IsClosed); err != nil {
		ctx.ServerError("UpdateMilestone", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.milestones.edit_success", m.Name))
	ctx.Redirect(ctx.Repo.RepoLink + "/milestones")
}

// ChangeMilestoneStatus response for change a milestone's status
func ChangeMilestoneStatus(ctx *context.Context) {
	var toClose bool
	switch ctx.Params(":action") {
	case "open":
		toClose = false
	case "close":
		toClose = true
	default:
		ctx.JSONRedirect(ctx.Repo.RepoLink + "/milestones")
		return
	}
	id := ctx.ParamsInt64(":id")

	if err := issues_model.ChangeMilestoneStatusByRepoIDAndID(ctx, ctx.Repo.Repository.ID, id, toClose); err != nil {
		if issues_model.IsErrMilestoneNotExist(err) {
			ctx.NotFound("", err)
		} else {
			ctx.ServerError("ChangeMilestoneStatusByIDAndRepoID", err)
		}
		return
	}
	ctx.JSONRedirect(ctx.Repo.RepoLink + "/milestones?state=" + url.QueryEscape(ctx.Params(":action")))
}

// DeleteMilestone delete a milestone
func DeleteMilestone(ctx *context.Context) {
	if err := issues_model.DeleteMilestoneByRepoID(ctx, ctx.Repo.Repository.ID, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteMilestoneByRepoID: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("repo.milestones.deletion_success"))
	}

	ctx.JSONRedirect(ctx.Repo.RepoLink + "/milestones")
}

// MilestoneIssuesAndPulls lists all the issues and pull requests of the milestone
func MilestoneIssuesAndPulls(ctx *context.Context) {
	milestoneID := ctx.ParamsInt64(":id")
	projectID := ctx.FormInt64("project")
	milestone, err := issues_model.GetMilestoneByRepoID(ctx, ctx.Repo.Repository.ID, milestoneID)
	if err != nil {
		if issues_model.IsErrMilestoneNotExist(err) {
			ctx.NotFound("GetMilestoneByID", err)
			return
		}

		ctx.ServerError("GetMilestoneByID", err)
		return
	}

	milestone.RenderedContent, err = markdown.RenderString(&markup.RenderContext{
		Links: markup.Links{
			Base: ctx.Repo.RepoLink,
		},
		Metas:   ctx.Repo.Repository.ComposeMetas(ctx),
		GitRepo: ctx.Repo.GitRepo,
		Ctx:     ctx,
	}, milestone.Content)
	if err != nil {
		ctx.ServerError("RenderString", err)
		return
	}

	ctx.Data["Title"] = milestone.Name
	ctx.Data["Milestone"] = milestone

	issues(ctx, milestoneID, projectID, optional.None[bool]())

	ret, _ := issue.GetTemplatesFromDefaultBranch(ctx.Repo.Repository, ctx.Repo.GitRepo)
	ctx.Data["NewIssueChooseTemplate"] = len(ret) > 0

	ctx.Data["CanWriteIssues"] = ctx.Repo.CanWriteIssuesOrPulls(false)
	ctx.Data["CanWritePulls"] = ctx.Repo.CanWriteIssuesOrPulls(true)

	ctx.HTML(http.StatusOK, tplMilestoneIssues)
}
