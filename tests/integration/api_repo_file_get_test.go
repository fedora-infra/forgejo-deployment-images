// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/modules/git"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIGetRawFileOrLFS(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Test with raw file
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/media/README.md")
	resp := MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, "# repo1\n\nDescription for repo1", resp.Body.String())

	// Test with LFS
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		httpContext := NewAPITestContext(t, "user2", "repo-lfs-test", auth_model.AccessTokenScopeWriteRepository)
		doAPICreateRepository(httpContext, false, git.Sha1ObjectFormat, func(t *testing.T, repository api.Repository) { // FIXME: use forEachObjectFormat
			u.Path = httpContext.GitPath()
			dstPath := t.TempDir()

			u.Path = httpContext.GitPath()
			u.User = url.UserPassword("user2", userPassword)

			t.Run("Clone", doGitClone(dstPath, u))

			dstPath2 := t.TempDir()

			t.Run("Partial Clone", doPartialGitClone(dstPath2, u))

			lfs, _ := lfsCommitAndPushTest(t, dstPath)

			reqLFS := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/media/"+lfs)
			respLFS := MakeRequestNilResponseRecorder(t, reqLFS, http.StatusOK)
			assert.Equal(t, littleSize, respLFS.Length)

			doAPIDeleteRepository(httpContext)
		})
	})
}
