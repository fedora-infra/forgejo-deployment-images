// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
)

func TestRepoWatch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// Test round-trip auto-watch
		setting.Service.AutoWatchOnChanges = true
		session := loginUser(t, "user2")
		unittest.AssertNotExistsBean(t, &repo_model.Watch{UserID: 2, RepoID: 3})
		testEditFile(t, session, "org3", "repo3", "master", "README.md", "Hello, World (Edited for watch)\n")
		unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{UserID: 2, RepoID: 3, Mode: repo_model.WatchModeAuto})
	})
}
