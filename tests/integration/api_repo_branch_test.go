// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIRepoBranchesPlain(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		session := loginUser(t, user1.LowerName)

		// public only token should be forbidden
		publicOnlyToken := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopePublicOnly, auth_model.AccessTokenScopeWriteRepository)
		link, _ := url.Parse(fmt.Sprintf("/api/v1/repos/org3/%s/branches", repo3.Name)) // a plain repo
		MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(publicOnlyToken), http.StatusForbidden)

		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		resp := MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
		bs, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var branches []*api.Branch
		require.NoError(t, json.Unmarshal(bs, &branches))
		assert.Len(t, branches, 2)
		assert.EqualValues(t, "test_branch", branches[0].Name)
		assert.EqualValues(t, "master", branches[1].Name)

		link2, _ := url.Parse(fmt.Sprintf("/api/v1/repos/org3/%s/branches/test_branch", repo3.Name))
		MakeRequest(t, NewRequest(t, "GET", link2.String()).AddTokenAuth(publicOnlyToken), http.StatusForbidden)

		resp = MakeRequest(t, NewRequest(t, "GET", link2.String()).AddTokenAuth(token), http.StatusOK)
		bs, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		var branch api.Branch
		require.NoError(t, json.Unmarshal(bs, &branch))
		assert.EqualValues(t, "test_branch", branch.Name)

		MakeRequest(t, NewRequest(t, "POST", link.String()).AddTokenAuth(publicOnlyToken), http.StatusForbidden)

		req := NewRequest(t, "POST", link.String()).AddTokenAuth(token)
		req.Header.Add("Content-Type", "application/json")
		req.Body = io.NopCloser(bytes.NewBufferString(`{"new_branch_name":"test_branch2", "old_branch_name": "test_branch", "old_ref_name":"refs/heads/test_branch"}`))
		resp = MakeRequest(t, req, http.StatusCreated)
		bs, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		var branch2 api.Branch
		require.NoError(t, json.Unmarshal(bs, &branch2))
		assert.EqualValues(t, "test_branch2", branch2.Name)
		assert.EqualValues(t, branch.Commit.ID, branch2.Commit.ID)

		resp = MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
		bs, err = io.ReadAll(resp.Body)
		require.NoError(t, err)

		branches = []*api.Branch{}
		require.NoError(t, json.Unmarshal(bs, &branches))
		assert.Len(t, branches, 3)
		assert.EqualValues(t, "test_branch", branches[0].Name)
		assert.EqualValues(t, "test_branch2", branches[1].Name)
		assert.EqualValues(t, "master", branches[2].Name)

		link3, _ := url.Parse(fmt.Sprintf("/api/v1/repos/org3/%s/branches/test_branch2", repo3.Name))
		MakeRequest(t, NewRequest(t, "DELETE", link3.String()), http.StatusNotFound)
		MakeRequest(t, NewRequest(t, "DELETE", link3.String()).AddTokenAuth(publicOnlyToken), http.StatusForbidden)

		MakeRequest(t, NewRequest(t, "DELETE", link3.String()).AddTokenAuth(token), http.StatusNoContent)
		require.NoError(t, err)
	})
}

func TestAPIRepoBranchesMirror(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo5 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 5})
	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	session := loginUser(t, user1.LowerName)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	link, _ := url.Parse(fmt.Sprintf("/api/v1/repos/org3/%s/branches", repo5.Name)) // a mirror repo
	resp := MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
	bs, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var branches []*api.Branch
	require.NoError(t, json.Unmarshal(bs, &branches))
	assert.Len(t, branches, 2)
	assert.EqualValues(t, "test_branch", branches[0].Name)
	assert.EqualValues(t, "master", branches[1].Name)

	link2, _ := url.Parse(fmt.Sprintf("/api/v1/repos/org3/%s/branches/test_branch", repo5.Name))
	resp = MakeRequest(t, NewRequest(t, "GET", link2.String()).AddTokenAuth(token), http.StatusOK)
	bs, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	var branch api.Branch
	require.NoError(t, json.Unmarshal(bs, &branch))
	assert.EqualValues(t, "test_branch", branch.Name)

	req := NewRequest(t, "POST", link.String()).AddTokenAuth(token)
	req.Header.Add("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewBufferString(`{"new_branch_name":"test_branch2", "old_branch_name": "test_branch", "old_ref_name":"refs/heads/test_branch"}`))
	resp = MakeRequest(t, req, http.StatusForbidden)
	bs, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.EqualValues(t, "{\"message\":\"Git Repository is a mirror.\",\"url\":\""+setting.AppURL+"api/swagger\"}\n", string(bs))

	resp = MakeRequest(t, NewRequest(t, "DELETE", link2.String()).AddTokenAuth(token), http.StatusForbidden)
	bs, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.EqualValues(t, "{\"message\":\"Git Repository is a mirror.\",\"url\":\""+setting.AppURL+"api/swagger\"}\n", string(bs))
}
