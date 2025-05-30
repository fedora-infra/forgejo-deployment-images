// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	git_model "forgejo.org/models/git"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func testAPIGetBranch(t *testing.T, branchName string, exists bool) {
	token := getUserToken(t, "user2", auth_model.AccessTokenScopeReadRepository)
	req := NewRequestf(t, "GET", "/api/v1/repos/user2/repo1/branches/%s", branchName).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, NoExpectedStatus)
	if !exists {
		assert.EqualValues(t, http.StatusNotFound, resp.Code)
		return
	}
	assert.EqualValues(t, http.StatusOK, resp.Code)
	var branch api.Branch
	DecodeJSON(t, resp, &branch)
	assert.EqualValues(t, branchName, branch.Name)
	assert.True(t, branch.UserCanPush)
	assert.True(t, branch.UserCanMerge)
}

func testAPIGetBranchProtection(t *testing.T, branchName string, expectedHTTPStatus int) *api.BranchProtection {
	token := getUserToken(t, "user2", auth_model.AccessTokenScopeReadRepository)
	req := NewRequestf(t, "GET", "/api/v1/repos/user2/repo1/branch_protections/%s", branchName).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, expectedHTTPStatus)

	if resp.Code == http.StatusOK {
		var branchProtection api.BranchProtection
		DecodeJSON(t, resp, &branchProtection)
		assert.EqualValues(t, branchName, branchProtection.RuleName)
		return &branchProtection
	}
	return nil
}

func testAPICreateBranchProtection(t *testing.T, branchName string, expectedHTTPStatus int) {
	token := getUserToken(t, "user2", auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/branch_protections", &api.BranchProtection{
		RuleName: branchName,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, expectedHTTPStatus)

	if resp.Code == http.StatusCreated {
		var branchProtection api.BranchProtection
		DecodeJSON(t, resp, &branchProtection)
		assert.EqualValues(t, branchName, branchProtection.RuleName)
	}
}

func testAPIEditBranchProtection(t *testing.T, branchName string, body *api.BranchProtection, expectedHTTPStatus int) {
	token := getUserToken(t, "user2", auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, "PATCH", "/api/v1/repos/user2/repo1/branch_protections/"+branchName, body).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, expectedHTTPStatus)

	if resp.Code == http.StatusOK {
		var branchProtection api.BranchProtection
		DecodeJSON(t, resp, &branchProtection)
		assert.EqualValues(t, branchName, branchProtection.RuleName)
	}
}

func testAPIDeleteBranchProtection(t *testing.T, branchName string, expectedHTTPStatus int) {
	token := getUserToken(t, "user2", auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestf(t, "DELETE", "/api/v1/repos/user2/repo1/branch_protections/%s", branchName).
		AddTokenAuth(token)
	MakeRequest(t, req, expectedHTTPStatus)
}

func testAPIDeleteBranch(t *testing.T, branchName string, expectedHTTPStatus int) {
	token := getUserToken(t, "user2", auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestf(t, "DELETE", "/api/v1/repos/user2/repo1/branches/%s", branchName).
		AddTokenAuth(token)
	MakeRequest(t, req, expectedHTTPStatus)
}

func TestAPIGetBranch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	for _, test := range []struct {
		BranchName string
		Exists     bool
	}{
		{"master", true},
		{"master/doesnotexist", false},
		{"feature/1", true},
		{"feature/1/doesnotexist", false},
	} {
		testAPIGetBranch(t, test.BranchName, test.Exists)
	}
}

func TestAPICreateBranch(t *testing.T) {
	onGiteaRun(t, testAPICreateBranches)
}

func testAPICreateBranches(t *testing.T, giteaURL *url.URL) {
	forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
		ctx := NewAPITestContext(t, "user2", "my-noo-repo-"+objectFormat.Name(), auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		giteaURL.Path = ctx.GitPath()

		t.Run("CreateRepo", doAPICreateRepository(ctx, false, objectFormat))
		testCases := []struct {
			OldBranch          string
			NewBranch          string
			ExpectedHTTPStatus int
		}{
			// Creating branch from default branch
			{
				OldBranch:          "",
				NewBranch:          "new_branch_from_default_branch",
				ExpectedHTTPStatus: http.StatusCreated,
			},
			// Creating branch from master
			{
				OldBranch:          "master",
				NewBranch:          "new_branch_from_master_1",
				ExpectedHTTPStatus: http.StatusCreated,
			},
			// Trying to create from master but already exists
			{
				OldBranch:          "master",
				NewBranch:          "new_branch_from_master_1",
				ExpectedHTTPStatus: http.StatusConflict,
			},
			// Trying to create from other branch (not default branch)
			// ps: it can't test the case-sensitive behavior here: the "BRANCH_2" can't be created by git on a case-insensitive filesystem, it makes the test fail quickly before the database code.
			// Suppose some users are running Gitea on a case-insensitive filesystem, it seems that it's unable to support case-sensitive branch names.
			{
				OldBranch:          "new_branch_from_master_1",
				NewBranch:          "branch_2",
				ExpectedHTTPStatus: http.StatusCreated,
			},
			// Trying to create from a branch which does not exist
			{
				OldBranch:          "does_not_exist",
				NewBranch:          "new_branch_from_non_existent",
				ExpectedHTTPStatus: http.StatusNotFound,
			},
			// Trying to create a branch with UTF8
			{
				OldBranch:          "master",
				NewBranch:          "test-👀",
				ExpectedHTTPStatus: http.StatusCreated,
			},
		}
		for _, test := range testCases {
			session := ctx.Session
			t.Run(test.NewBranch, func(t *testing.T) {
				testAPICreateBranch(t, session, ctx.Username, ctx.Reponame, test.OldBranch, test.NewBranch, test.ExpectedHTTPStatus)
			})
		}
	})
}

func testAPICreateBranch(t testing.TB, session *TestSession, user, repo, oldBranch, newBranch string, status int) bool {
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/"+user+"/"+repo+"/branches", &api.CreateBranchRepoOption{
		BranchName:    newBranch,
		OldBranchName: oldBranch,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, status)

	var branch api.Branch
	DecodeJSON(t, resp, &branch)

	if resp.Result().StatusCode == http.StatusCreated {
		assert.EqualValues(t, newBranch, branch.Name)
	}

	return resp.Result().StatusCode == status
}

func TestAPIUpdateBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		t.Run("UpdateBranchWithEmptyRepo", func(t *testing.T) {
			testAPIUpdateBranch(t, "user10", "repo6", "master", "test", http.StatusNotFound)
		})
		t.Run("UpdateBranchWithSameBranchNames", func(t *testing.T) {
			resp := testAPIUpdateBranch(t, "user2", "repo1", "master", "master", http.StatusUnprocessableEntity)
			assert.Contains(t, resp.Body.String(), "Cannot rename a branch using the same name or rename to a branch that already exists.")
		})
		t.Run("UpdateBranchThatAlreadyExists", func(t *testing.T) {
			resp := testAPIUpdateBranch(t, "user2", "repo1", "master", "branch2", http.StatusUnprocessableEntity)
			assert.Contains(t, resp.Body.String(), "Cannot rename a branch using the same name or rename to a branch that already exists.")
		})
		t.Run("UpdateBranchWithNonExistentBranch", func(t *testing.T) {
			resp := testAPIUpdateBranch(t, "user2", "repo1", "i-dont-exist", "new-branch-name", http.StatusNotFound)
			assert.Contains(t, resp.Body.String(), "Branch doesn't exist.")
		})
		t.Run("RenameBranchNormalScenario", func(t *testing.T) {
			testAPIUpdateBranch(t, "user2", "repo1", "branch2", "new-branch-name", http.StatusNoContent)
		})
	})
}

func testAPIUpdateBranch(t *testing.T, ownerName, repoName, from, to string, expectedHTTPStatus int) *httptest.ResponseRecorder {
	token := getUserToken(t, ownerName, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, "PATCH", "api/v1/repos/"+ownerName+"/"+repoName+"/branches/"+from, &api.UpdateBranchRepoOption{
		Name: to,
	}).AddTokenAuth(token)
	return MakeRequest(t, req, expectedHTTPStatus)
}

func TestAPIBranchProtection(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Branch protection  on branch that not exist
	testAPICreateBranchProtection(t, "master/doesnotexist", http.StatusCreated)
	// Get branch protection on branch that exist but not branch protection
	testAPIGetBranchProtection(t, "master", http.StatusNotFound)

	testAPICreateBranchProtection(t, "master", http.StatusCreated)
	// Can only create once
	testAPICreateBranchProtection(t, "master", http.StatusForbidden)

	// Can't delete a protected branch
	testAPIDeleteBranch(t, "master", http.StatusForbidden)

	testAPIGetBranchProtection(t, "master", http.StatusOK)
	testAPIEditBranchProtection(t, "master", &api.BranchProtection{
		EnablePush: true,
	}, http.StatusOK)

	// enable status checks, require the "test1" check to pass
	testAPIEditBranchProtection(t, "master", &api.BranchProtection{
		EnableStatusCheck:   true,
		StatusCheckContexts: []string{"test1"},
	}, http.StatusOK)
	bp := testAPIGetBranchProtection(t, "master", http.StatusOK)
	assert.True(t, bp.EnableStatusCheck)
	assert.Equal(t, []string{"test1"}, bp.StatusCheckContexts)

	// disable status checks, clear the list of required checks
	testAPIEditBranchProtection(t, "master", &api.BranchProtection{
		EnableStatusCheck:   false,
		StatusCheckContexts: []string{},
	}, http.StatusOK)
	bp = testAPIGetBranchProtection(t, "master", http.StatusOK)
	assert.False(t, bp.EnableStatusCheck)
	assert.Equal(t, []string{}, bp.StatusCheckContexts)

	testAPIDeleteBranchProtection(t, "master", http.StatusNoContent)

	// Test branch deletion
	testAPIDeleteBranch(t, "master", http.StatusForbidden)
	testAPIDeleteBranch(t, "branch2", http.StatusNoContent)
}

func TestAPICreateBranchWithSyncBranches(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		unittest.AssertCount(t, &git_model.Branch{RepoID: 1}, 4)

		// make a broke repository with no branch on database
		unittest.AssertSuccessfulDelete(t, &git_model.Branch{RepoID: 1})

		ctx := NewAPITestContext(t, "user2", "repo1", auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		giteaURL.Path = ctx.GitPath()

		testAPICreateBranch(t, ctx.Session, "user2", "repo1", "", "new_branch", http.StatusCreated)

		unittest.AssertExistsIf(t, true, &git_model.Branch{RepoID: 1, Name: "new_branch"})
	})
}
