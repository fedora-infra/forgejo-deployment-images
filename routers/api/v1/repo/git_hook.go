// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"forgejo.org/modules/git"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// ListGitHooks list all Git hooks of a repository
func ListGitHooks(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/hooks/git repository repoListGitHooks
	// ---
	// summary: List the Git hooks in a repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/GitHookList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	hooks, err := ctx.Repo.GitRepo.Hooks()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "Hooks", err)
		return
	}

	apiHooks := make([]*api.GitHook, len(hooks))
	for i := range hooks {
		apiHooks[i] = convert.ToGitHook(hooks[i])
	}
	ctx.JSON(http.StatusOK, &apiHooks)
}

// GetGitHook get a repo's Git hook by id
func GetGitHook(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/hooks/git/{id} repository repoGetGitHook
	// ---
	// summary: Get a Git hook
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the hook to get
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/GitHook"
	//   "404":
	//     "$ref": "#/responses/notFound"

	hookID := ctx.Params(":id")
	hook, err := ctx.Repo.GitRepo.GetHook(hookID)
	if err != nil {
		if err == git.ErrNotValidHook {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "GetHook", err)
		}
		return
	}
	ctx.JSON(http.StatusOK, convert.ToGitHook(hook))
}

// EditGitHook modify a Git hook of a repository
func EditGitHook(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/hooks/git/{id} repository repoEditGitHook
	// ---
	// summary: Edit a Git hook in a repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the hook to get
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditGitHookOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/GitHook"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*api.EditGitHookOption)
	hookID := ctx.Params(":id")
	hook, err := ctx.Repo.GitRepo.GetHook(hookID)
	if err != nil {
		if err == git.ErrNotValidHook {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "GetHook", err)
		}
		return
	}

	hook.Content = form.Content
	if err = hook.Update(); err != nil {
		ctx.Error(http.StatusInternalServerError, "hook.Update", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToGitHook(hook))
}

// DeleteGitHook delete a Git hook of a repository
func DeleteGitHook(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/hooks/git/{id} repository repoDeleteGitHook
	// ---
	// summary: Delete a Git hook in a repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the hook to get
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	hookID := ctx.Params(":id")
	hook, err := ctx.Repo.GitRepo.GetHook(hookID)
	if err != nil {
		if err == git.ErrNotValidHook {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "GetHook", err)
		}
		return
	}

	hook.Content = ""
	if err = hook.Update(); err != nil {
		ctx.Error(http.StatusInternalServerError, "hook.Update", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
