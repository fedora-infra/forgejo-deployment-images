// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"forgejo.org/services/context"
	files_service "forgejo.org/services/repository/files"
)

// GetBlob get the blob of a repository file.
func GetBlob(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/git/blobs/{sha} repository GetBlob
	// ---
	// summary: Gets the blob of a repository.
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
	// - name: sha
	//   in: path
	//   description: sha of the commit
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/GitBlobResponse"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	sha := ctx.Params("sha")
	if len(sha) == 0 {
		ctx.Error(http.StatusBadRequest, "", "sha not provided")
		return
	}

	if blob, err := files_service.GetBlobBySHA(ctx, ctx.Repo.Repository, ctx.Repo.GitRepo, sha); err != nil {
		ctx.Error(http.StatusBadRequest, "", err)
	} else {
		ctx.JSON(http.StatusOK, blob)
	}
}
