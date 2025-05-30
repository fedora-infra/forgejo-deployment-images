// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"

	git_model "forgejo.org/models/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/routers/web/repo"
	"forgejo.org/services/context"
	repo_service "forgejo.org/services/repository"
)

// SetDefaultBranchPost set default branch
func SetDefaultBranchPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.branches.update_default_branch")
	ctx.Data["PageIsSettingsBranches"] = true

	repo.PrepareBranchList(ctx)
	if ctx.Written() {
		return
	}

	repo := ctx.Repo.Repository

	switch ctx.FormString("action") {
	case "default_branch":
		if ctx.HasError() {
			ctx.HTML(http.StatusOK, tplBranches)
			return
		}

		branch := ctx.FormString("branch")
		if err := repo_service.SetRepoDefaultBranch(ctx, ctx.Repo.Repository, ctx.Repo.GitRepo, branch); err != nil {
			switch {
			case git_model.IsErrBranchNotExist(err):
				ctx.Status(http.StatusNotFound)
			default:
				ctx.ServerError("SetDefaultBranch", err)
			}
			return
		}

		log.Trace("Repository basic settings updated: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
	default:
		ctx.NotFound("", nil)
	}
}
