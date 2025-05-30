// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIAdminOrgCreate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteAdmin)

	org := api.CreateOrgOption{
		UserName:    "user2_org",
		FullName:    "User2's organization",
		Description: "This organization created by admin for user2",
		Website:     "https://try.gitea.io",
		Location:    "Shanghai",
		Visibility:  "private",
	}
	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/users/user2/orgs", &org).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var apiOrg api.Organization
	DecodeJSON(t, resp, &apiOrg)

	assert.Equal(t, org.UserName, apiOrg.Name)
	assert.Equal(t, org.FullName, apiOrg.FullName)
	assert.Equal(t, org.Description, apiOrg.Description)
	assert.Equal(t, org.Website, apiOrg.Website)
	assert.Equal(t, org.Location, apiOrg.Location)
	assert.Equal(t, org.Visibility, apiOrg.Visibility)

	unittest.AssertExistsAndLoadBean(t, &user_model.User{
		Name:      org.UserName,
		LowerName: strings.ToLower(org.UserName),
		FullName:  org.FullName,
	})
}

func TestAPIAdminOrgCreateBadVisibility(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteAdmin)

	org := api.CreateOrgOption{
		UserName:    "user2_org",
		FullName:    "User2's organization",
		Description: "This organization created by admin for user2",
		Website:     "https://try.gitea.io",
		Location:    "Shanghai",
		Visibility:  "notvalid",
	}
	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/users/user2/orgs", &org).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

func TestAPIAdminOrgCreateNotAdmin(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	nonAdminUsername := "user2"
	session := loginUser(t, nonAdminUsername)
	token := getTokenForLoggedInUser(t, session)
	org := api.CreateOrgOption{
		UserName:    "user2_org",
		FullName:    "User2's organization",
		Description: "This organization created by admin for user2",
		Website:     "https://try.gitea.io",
		Location:    "Shanghai",
		Visibility:  "public",
	}
	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/users/user2/orgs", &org).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)
}
