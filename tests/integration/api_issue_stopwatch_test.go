// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIListStopWatches(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository, auth_model.AccessTokenScopeReadUser)
	req := NewRequest(t, "GET", "/api/v1/user/stopwatches").
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	var apiWatches []*api.StopWatch
	DecodeJSON(t, resp, &apiWatches)
	stopwatch := unittest.AssertExistsAndLoadBean(t, &issues_model.Stopwatch{UserID: owner.ID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: stopwatch.IssueID})
	if assert.Len(t, apiWatches, 1) {
		assert.EqualValues(t, stopwatch.CreatedUnix.AsTime().Unix(), apiWatches[0].Created.Unix())
		assert.EqualValues(t, issue.Index, apiWatches[0].IssueIndex)
		assert.EqualValues(t, issue.Title, apiWatches[0].IssueTitle)
		assert.EqualValues(t, repo.Name, apiWatches[0].RepoName)
		assert.EqualValues(t, repo.OwnerName, apiWatches[0].RepoOwnerName)
		assert.Positive(t, apiWatches[0].Seconds)
	}
}

func TestAPIStopStopWatches(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	_ = issue.LoadRepo(db.DefaultContext)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: issue.Repo.OwnerID})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	req := NewRequestf(t, "POST", "/api/v1/repos/%s/%s/issues/%d/stopwatch/stop", owner.Name, issue.Repo.Name, issue.Index).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)
	MakeRequest(t, req, http.StatusConflict)
}

func TestAPICancelStopWatches(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	_ = issue.LoadRepo(db.DefaultContext)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: issue.Repo.OwnerID})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/issues/%d/stopwatch/delete", owner.Name, issue.Repo.Name, issue.Index).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
	MakeRequest(t, req, http.StatusConflict)
}

func TestAPIStartStopWatches(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 3})
	_ = issue.LoadRepo(db.DefaultContext)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: issue.Repo.OwnerID})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	req := NewRequestf(t, "POST", "/api/v1/repos/%s/%s/issues/%d/stopwatch/start", owner.Name, issue.Repo.Name, issue.Index).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)
	MakeRequest(t, req, http.StatusConflict)
}
