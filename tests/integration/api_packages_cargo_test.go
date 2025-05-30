// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/json"
	cargo_module "forgejo.org/modules/packages/cargo"
	"forgejo.org/modules/setting"
	cargo_router "forgejo.org/routers/api/packages/cargo"
	gitea_context "forgejo.org/services/context"
	cargo_service "forgejo.org/services/packages/cargo"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageCargo(t *testing.T) {
	onGiteaRun(t, testPackageCargo)
}

func testPackageCargo(t *testing.T, _ *neturl.URL) {
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	packageName := "cargo-package"
	packageVersion := "1.0.3"
	packageDescription := "Package Description"
	packageAuthor := "KN4CK3R"
	packageHomepage := "https://gitea.io/"
	packageLicense := "MIT"

	createPackage := func(name, version string) io.Reader {
		metadata := `{
   "name":"` + name + `",
   "vers":"` + version + `",
   "description":"` + packageDescription + `",
   "authors": ["` + packageAuthor + `"],
   "deps":[
      {
         "name":"dep",
         "version_req":"1.0",
         "registry": "https://gitea.io/user/_cargo-index",
         "kind": "normal",
         "default_features": true
      }
   ],
   "homepage":"` + packageHomepage + `",
   "license":"` + packageLicense + `"
}`

		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, uint32(len(metadata)))
		buf.WriteString(metadata)
		binary.Write(&buf, binary.LittleEndian, uint32(4))
		buf.WriteString("test")
		return &buf
	}

	err := cargo_service.InitializeIndexRepository(db.DefaultContext, user, user)
	require.NoError(t, err)

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, user.Name, cargo_service.IndexRepositoryName)
	assert.NotNil(t, repo)
	require.NoError(t, err)

	readGitContent := func(t *testing.T, path string) string {
		gitRepo, err := gitrepo.OpenRepository(db.DefaultContext, repo)
		require.NoError(t, err)
		defer gitRepo.Close()

		commit, err := gitRepo.GetBranchCommit(repo.DefaultBranch)
		require.NoError(t, err)

		blob, err := commit.GetBlobByPath(path)
		require.NoError(t, err)

		content, err := blob.GetBlobContent(1024)
		require.NoError(t, err)

		return content
	}

	root := fmt.Sprintf("%sapi/packages/%s/cargo", setting.AppURL, user.Name)
	url := fmt.Sprintf("%s/api/v1/crates", root)

	t.Run("Index", func(t *testing.T) {
		t.Run("Git/Config", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			content := readGitContent(t, cargo_service.ConfigFileName)

			var config cargo_service.Config
			err := json.Unmarshal([]byte(content), &config)
			require.NoError(t, err)

			assert.Equal(t, url, config.DownloadURL)
			assert.Equal(t, root, config.APIURL)
		})

		t.Run("HTTP/Config", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", root+"/"+cargo_service.ConfigFileName)
			resp := MakeRequest(t, req, http.StatusOK)

			var config cargo_service.Config
			err := json.Unmarshal(resp.Body.Bytes(), &config)
			require.NoError(t, err)

			assert.Equal(t, url, config.DownloadURL)
			assert.Equal(t, root, config.APIURL)
		})
	})

	t.Run("Upload", func(t *testing.T) {
		t.Run("InvalidNameOrVersion", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			content := createPackage("0test", "1.0.0")

			req := NewRequestWithBody(t, "PUT", url+"/new", content).
				AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusBadRequest)

			var status cargo_router.StatusResponse
			DecodeJSON(t, resp, &status)
			assert.False(t, status.OK)

			content = createPackage("test", "-1.0.0")

			req = NewRequestWithBody(t, "PUT", url+"/new", content).
				AddBasicAuth(user.Name)
			resp = MakeRequest(t, req, http.StatusBadRequest)

			DecodeJSON(t, resp, &status)
			assert.False(t, status.OK)
		})

		t.Run("InvalidContent", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			metadata := `{"name":"test","vers":"1.0.0"}`

			var buf bytes.Buffer
			binary.Write(&buf, binary.LittleEndian, uint32(len(metadata)))
			buf.WriteString(metadata)
			binary.Write(&buf, binary.LittleEndian, uint32(4))
			buf.WriteString("te")

			req := NewRequestWithBody(t, "PUT", url+"/new", &buf).
				AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		t.Run("Valid", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, "PUT", url+"/new", createPackage(packageName, packageVersion))
			MakeRequest(t, req, http.StatusUnauthorized)

			req = NewRequestWithBody(t, "PUT", url+"/new", createPackage(packageName, packageVersion)).
				AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusOK)

			var status cargo_router.StatusResponse
			DecodeJSON(t, resp, &status)
			assert.True(t, status.OK)

			pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeCargo)
			require.NoError(t, err)
			assert.Len(t, pvs, 1)

			pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
			require.NoError(t, err)
			assert.NotNil(t, pd.SemVer)
			assert.IsType(t, &cargo_module.Metadata{}, pd.Metadata)
			assert.Equal(t, packageName, pd.Package.Name)
			assert.Equal(t, packageVersion, pd.Version.Version)

			pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
			require.NoError(t, err)
			assert.Len(t, pfs, 1)
			assert.Equal(t, fmt.Sprintf("%s-%s.crate", packageName, packageVersion), pfs[0].Name)
			assert.True(t, pfs[0].IsLead)

			pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
			require.NoError(t, err)
			assert.EqualValues(t, 4, pb.Size)

			req = NewRequestWithBody(t, "PUT", url+"/new", createPackage(packageName, packageVersion)).
				AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusConflict)

			t.Run("Index", func(t *testing.T) {
				t.Run("Git", func(t *testing.T) {
					t.Run("Entry", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						content := readGitContent(t, cargo_service.BuildPackagePath(packageName))

						var entry cargo_service.IndexVersionEntry
						err := json.Unmarshal([]byte(content), &entry)
						require.NoError(t, err)

						assert.Equal(t, packageName, entry.Name)
						assert.Equal(t, packageVersion, entry.Version)
						assert.Equal(t, pb.HashSHA256, entry.FileChecksum)
						assert.False(t, entry.Yanked)
						assert.Len(t, entry.Dependencies, 1)
						dep := entry.Dependencies[0]
						assert.Equal(t, "dep", dep.Name)
						assert.Equal(t, "1.0", dep.Req)
						assert.Equal(t, "normal", dep.Kind)
						assert.True(t, dep.DefaultFeatures)
						assert.Empty(t, dep.Features)
						assert.False(t, dep.Optional)
						assert.Nil(t, dep.Target)
						assert.NotNil(t, dep.Registry)
						assert.Equal(t, "https://gitea.io/user/_cargo-index", *dep.Registry)
						assert.Nil(t, dep.Package)
					})

					t.Run("Rebuild", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						err := cargo_service.RebuildIndex(db.DefaultContext, user, user)
						require.NoError(t, err)

						_ = readGitContent(t, cargo_service.BuildPackagePath(packageName))
					})
				})

				t.Run("HTTP", func(t *testing.T) {
					t.Run("Entry", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequest(t, "GET", root+"/"+cargo_service.BuildPackagePath(packageName))
						resp := MakeRequest(t, req, http.StatusOK)

						var entry cargo_service.IndexVersionEntry
						err := json.Unmarshal(resp.Body.Bytes(), &entry)
						require.NoError(t, err)

						assert.Equal(t, packageName, entry.Name)
						assert.Equal(t, packageVersion, entry.Version)
						assert.Equal(t, pb.HashSHA256, entry.FileChecksum)
						assert.False(t, entry.Yanked)
						assert.Len(t, entry.Dependencies, 1)
						dep := entry.Dependencies[0]
						assert.Equal(t, "dep", dep.Name)
						assert.Equal(t, "1.0", dep.Req)
						assert.Equal(t, "normal", dep.Kind)
						assert.True(t, dep.DefaultFeatures)
						assert.Empty(t, dep.Features)
						assert.False(t, dep.Optional)
						assert.Nil(t, dep.Target)
						assert.NotNil(t, dep.Registry)
						assert.Equal(t, "https://gitea.io/user/_cargo-index", *dep.Registry)
						assert.Nil(t, dep.Package)
					})
				})
			})
		})
	})

	t.Run("Download", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		pv, err := packages.GetVersionByNameAndVersion(db.DefaultContext, user.ID, packages.TypeCargo, packageName, packageVersion)
		require.NoError(t, err)
		assert.EqualValues(t, 0, pv.DownloadCount)

		pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pv.ID)
		require.NoError(t, err)
		assert.Len(t, pfs, 1)

		req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s/download", url, neturl.PathEscape(packageName), neturl.PathEscape(pv.Version))).
			AddBasicAuth(user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, "test", resp.Body.String())

		pv, err = packages.GetVersionByNameAndVersion(db.DefaultContext, user.ID, packages.TypeCargo, packageName, packageVersion)
		require.NoError(t, err)
		assert.EqualValues(t, 1, pv.DownloadCount)
	})

	t.Run("Search", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		cases := []struct {
			Query           string
			Page            int
			PerPage         int
			ExpectedTotal   int64
			ExpectedResults int
		}{
			{"", 0, 0, 1, 1},
			{"", 1, 10, 1, 1},
			{"cargo", 1, 0, 1, 1},
			{"cargo", 1, 10, 1, 1},
			{"cargo", 2, 10, 1, 0},
			{"test", 0, 10, 0, 0},
		}

		for i, c := range cases {
			req := NewRequest(t, "GET", fmt.Sprintf("%s?q=%s&page=%d&per_page=%d", url, c.Query, c.Page, c.PerPage)).
				AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusOK)

			var result cargo_router.SearchResult
			DecodeJSON(t, resp, &result)

			assert.Equal(t, c.ExpectedTotal, result.Meta.Total, "case %d: unexpected total hits", i)
			assert.Len(t, result.Crates, c.ExpectedResults, "case %d: unexpected result count", i)
		}
	})

	t.Run("Yank", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "DELETE", fmt.Sprintf("%s/%s/%s/yank", url, neturl.PathEscape(packageName), neturl.PathEscape(packageVersion))).
			AddBasicAuth(user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		var status cargo_router.StatusResponse
		DecodeJSON(t, resp, &status)
		assert.True(t, status.OK)

		content := readGitContent(t, cargo_service.BuildPackagePath(packageName))

		var entry cargo_service.IndexVersionEntry
		err := json.Unmarshal([]byte(content), &entry)
		require.NoError(t, err)

		assert.True(t, entry.Yanked)
	})

	t.Run("Unyank", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "PUT", fmt.Sprintf("%s/%s/%s/unyank", url, neturl.PathEscape(packageName), neturl.PathEscape(packageVersion))).
			AddBasicAuth(user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		var status cargo_router.StatusResponse
		DecodeJSON(t, resp, &status)
		assert.True(t, status.OK)

		content := readGitContent(t, cargo_service.BuildPackagePath(packageName))

		var entry cargo_service.IndexVersionEntry
		err := json.Unmarshal([]byte(content), &entry)
		require.NoError(t, err)

		assert.False(t, entry.Yanked)
	})

	t.Run("ListOwners", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/owners", url, neturl.PathEscape(packageName)))
		resp := MakeRequest(t, req, http.StatusOK)

		var owners cargo_router.Owners
		DecodeJSON(t, resp, &owners)

		assert.Len(t, owners.Users, 1)
		assert.Equal(t, user.ID, owners.Users[0].ID)
		assert.Equal(t, user.Name, owners.Users[0].Login)
		assert.Equal(t, user.DisplayName(), owners.Users[0].Name)
	})
}

func TestRebuildCargo(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *neturl.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		session := loginUser(t, user.Name)
		unittest.AssertExistsIf(t, false, &repo_model.Repository{OwnerID: user.ID, Name: cargo_service.IndexRepositoryName})

		t.Run("No index", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user/settings/packages/cargo/rebuild", map[string]string{
				"_csrf": GetCSRF(t, session, "/user/settings/packages"),
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.EqualValues(t, "error%3DCannot%2Brebuild%252C%2Bno%2Bindex%2Bis%2Binitialized.", flashCookie.Value)
			unittest.AssertExistsIf(t, false, &repo_model.Repository{OwnerID: user.ID, Name: cargo_service.IndexRepositoryName})
		})

		t.Run("Initialize Cargo", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user/settings/packages")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, `form[action="/user/settings/packages/cargo/rebuild"]`, false)
			htmlDoc.AssertElement(t, `form[action="/user/settings/packages/cargo/initialize"]`, true)

			req = NewRequestWithValues(t, "POST", "/user/settings/packages/cargo/initialize", map[string]string{
				"_csrf": htmlDoc.GetCSRF(),
			})
			session.MakeRequest(t, req, http.StatusSeeOther)
			unittest.AssertExistsIf(t, true, &repo_model.Repository{OwnerID: user.ID, Name: cargo_service.IndexRepositoryName})

			req = NewRequest(t, "GET", "/user/settings/packages")
			resp = session.MakeRequest(t, req, http.StatusOK)
			htmlDoc = NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, `form[action="/user/settings/packages/cargo/rebuild"]`, true)
			htmlDoc.AssertElement(t, `form[action="/user/settings/packages/cargo/initialize"]`, false)
		})

		t.Run("With index", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user/settings/packages/cargo/rebuild", map[string]string{
				"_csrf": GetCSRF(t, session, "/user/settings/packages"),
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.EqualValues(t, "success%3DThe%2BCargo%2Bindex%2Bwas%2Bsuccessfully%2Brebuild.", flashCookie.Value)
		})
	})
}
