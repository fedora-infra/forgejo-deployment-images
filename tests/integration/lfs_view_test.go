// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/lfs"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// check that files stored in LFS render properly in the web UI
func TestLFSRender(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	// check that a markup file is flagged with "Stored in Git LFS" and shows its text
	t.Run("Markup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/CONTRIBUTING.md")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		fileInfo := doc.Find("div.file-info-entry").First().Text()
		assert.Contains(t, fileInfo, "Stored with Git LFS")

		content := doc.Find("div.file-view").Text()
		assert.Contains(t, content, "Testing documents in LFS")
	})

	// check that an image is flagged with "Stored in Git LFS" and renders inline
	t.Run("Image", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/jpeg.jpg")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		fileInfo := doc.Find("div.file-info-entry").First().Text()
		assert.Contains(t, fileInfo, "Stored with Git LFS")

		src, exists := doc.Find(".file-view img").Attr("src")
		assert.True(t, exists, "The image should be in an <img> tag")
		assert.Equal(t, "/user2/lfs/media/branch/master/jpeg.jpg", src, "The image should use the /media link because it's in LFS")
	})

	// check that a binary file is flagged with "Stored in Git LFS" and renders a /media/ link instead of a /raw/ link
	t.Run("Binary", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/crypt.bin")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		fileInfo := doc.Find("div.file-info-entry").First().Text()
		assert.Contains(t, fileInfo, "Stored with Git LFS")

		rawLink, exists := doc.Find("div.file-view > div.view-raw > a").Attr("href")
		assert.True(t, exists, "Download link should render instead of content because this is a binary file")
		assert.Equal(t, "/user2/lfs/media/branch/master/crypt.bin", rawLink, "The download link should use the proper /media link because it's in LFS")
	})

	// check that a directory with a README file shows its text
	t.Run("Readme", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/subdir")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		content := doc.Find("div.file-view").Text()
		assert.Contains(t, content, "Testing READMEs in LFS")
	})

	t.Run("/settings/lfs/pointers", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// visit /user2/lfs/settings/lfs/pointer
		req := NewRequest(t, "GET", "/user2/lfs/settings/lfs/pointers")
		resp := session.MakeRequest(t, req, http.StatusOK)

		// follow the first link to /user2/lfs/settings/lfs/find?oid=....
		filesTable := NewHTMLParser(t, resp.Body).doc.Find("#lfs-files-table")
		assert.Contains(t, filesTable.Text(), "Find commits")
		lfsFind := filesTable.Find(`.primary.button[href^="/user2"]`)
		assert.Positive(t, lfsFind.Length())
		lfsFindPath, exists := lfsFind.First().Attr("href")
		assert.True(t, exists)

		assert.Contains(t, lfsFindPath, "oid=")
		req = NewRequest(t, "GET", lfsFindPath)
		resp = session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body).doc
		assert.Equal(t, 1, doc.Find(`.sha.label[href="/user2/lfs/commit/73cf03db6ece34e12bf91e8853dc58f678f2f82d"]`).Length(), "could not find link to commit")
	})

	// check that an invalid lfs entry defaults to plaintext
	t.Run("Invalid", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/invalid")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		content := doc.Find("div.file-view").Text()
		assert.Contains(t, content, "oid sha256:9d178b5f15046343fd32f451df93acc2bdd9e6373be478b968e4cad6b6647351")
	})
}

// TestLFSLockView tests the LFS lock view on settings page of repositories
func TestLFSLockView(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // in org 3
	repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3}) // own by org 3
	session := loginUser(t, user2.Name)

	// create a lock
	lockPath := "test_lfs_lock_view.zip"
	lockID := ""
	{
		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/%s.git/info/lfs/locks", repo3.FullName()), map[string]string{"path": lockPath})
		req.Header.Set("Accept", lfs.AcceptHeader)
		req.Header.Set("Content-Type", lfs.MediaType)
		resp := session.MakeRequest(t, req, http.StatusCreated)
		lockResp := &api.LFSLockResponse{}
		DecodeJSON(t, resp, lockResp)
		lockID = lockResp.Lock.ID
	}
	defer func() {
		// release the lock
		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/%s.git/info/lfs/locks/%s/unlock", repo3.FullName(), lockID), map[string]string{})
		req.Header.Set("Accept", lfs.AcceptHeader)
		req.Header.Set("Content-Type", lfs.MediaType)
		session.MakeRequest(t, req, http.StatusOK)
	}()

	t.Run("owner name", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// make sure the display names are different, or the test is meaningless
		require.NoError(t, repo3.LoadOwner(t.Context()))
		require.NotEqual(t, user2.DisplayName(), repo3.Owner.DisplayName())

		req := NewRequest(t, "GET", fmt.Sprintf("/%s/settings/lfs/locks", repo3.FullName()))
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		tr := doc.Find("table#lfs-files-locks-table tbody tr")
		require.Equal(t, 1, tr.Length())

		td := tr.First().Find("td")
		require.Equal(t, 4, td.Length())

		// path
		assert.Equal(t, lockPath, strings.TrimSpace(td.Eq(0).Text()))
		// owner name
		assert.Equal(t, user2.DisplayName(), strings.TrimSpace(td.Eq(1).Text()))
	})
}
