// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"testing"

	"forgejo.org/models/unittest"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTreeBySHA(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "user2/repo1")
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	sha := ctx.Repo.Repository.DefaultBranch
	page := 1
	perPage := 10
	ctx.SetParams(":id", "1")
	ctx.SetParams(":sha", sha)

	tree, err := GetTreeBySHA(ctx, ctx.Repo.Repository, ctx.Repo.GitRepo, ctx.Params(":sha"), page, perPage, true)
	require.NoError(t, err)
	expectedTree := &api.GitTreeResponse{
		SHA: "65f1bf27bc3bf70f64657658635e66094edbcb4d",
		URL: "https://try.gitea.io/api/v1/repos/user2/repo1/git/trees/65f1bf27bc3bf70f64657658635e66094edbcb4d",
		Entries: []api.GitEntry{
			{
				Path: "README.md",
				Mode: "100644",
				Type: "blob",
				Size: 30,
				SHA:  "4b4851ad51df6a7d9f25c979345979eaeb5b349f",
				URL:  "https://try.gitea.io/api/v1/repos/user2/repo1/git/blobs/4b4851ad51df6a7d9f25c979345979eaeb5b349f",
			},
		},
		Truncated:  false,
		Page:       1,
		TotalCount: 1,
	}

	assert.EqualValues(t, expectedTree, tree)
}
