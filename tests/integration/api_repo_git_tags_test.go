// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIGitTags(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	// Login as User2.
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

	// Set up git config for the tagger
	_ = git.NewCommand(git.DefaultContext, "config", "user.name").AddDynamicArguments(user.Name).Run(&git.RunOpts{Dir: repo.RepoPath()})
	_ = git.NewCommand(git.DefaultContext, "config", "user.email").AddDynamicArguments(user.Email).Run(&git.RunOpts{Dir: repo.RepoPath()})

	gitRepo, _ := gitrepo.OpenRepository(git.DefaultContext, repo)
	defer gitRepo.Close()

	commit, _ := gitRepo.GetBranchCommit("master")
	lTagName := "lightweightTag"
	gitRepo.CreateTag(lTagName, commit.ID.String())

	aTagName := "annotatedTag"
	aTagMessage := "my annotated message"
	gitRepo.CreateAnnotatedTag(aTagName, aTagMessage, commit.ID.String())
	aTag, _ := gitRepo.GetTag(aTagName)

	// SHOULD work for annotated tags
	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/git/tags/%s", user.Name, repo.Name, aTag.ID.String()).
		AddTokenAuth(token)
	res := MakeRequest(t, req, http.StatusOK)

	var tag *api.AnnotatedTag
	DecodeJSON(t, res, &tag)

	assert.Equal(t, aTagName, tag.Tag)
	assert.Equal(t, aTag.ID.String(), tag.SHA)
	assert.Equal(t, commit.ID.String(), tag.Object.SHA)
	assert.Equal(t, aTagMessage+"\n", tag.Message)
	assert.Equal(t, user.Name, tag.Tagger.Name)
	assert.Equal(t, user.Email, tag.Tagger.Email)
	assert.Equal(t, util.URLJoin(repo.APIURL(), "git/tags", aTag.ID.String()), tag.URL)

	// Should NOT work for lightweight tags
	badReq := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/git/tags/%s", user.Name, repo.Name, commit.ID.String()).
		AddTokenAuth(token)
	MakeRequest(t, badReq, http.StatusBadRequest)
}

func TestAPIDeleteTagByName(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/repos/%s/%s/tags/delete-tag", owner.Name, repo.Name)).
		AddTokenAuth(token)
	_ = MakeRequest(t, req, http.StatusNoContent)

	// Make sure that actual releases can't be deleted outright
	createNewReleaseUsingAPI(t, token, owner, repo, "release-tag", "", "Release Tag", "test")

	req = NewRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/repos/%s/%s/tags/release-tag", owner.Name, repo.Name)).
		AddTokenAuth(token)
	_ = MakeRequest(t, req, http.StatusConflict)
}
