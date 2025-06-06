// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"

	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/json"
	api "forgejo.org/modules/structs"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlackPayload(t *testing.T) {
	sc := slackConvertor{}

	t.Run("Create", func(t *testing.T) {
		p := createTestPayload()

		pl, err := sc.Create(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo:<http://localhost:3000/test/repo/src/branch/test|test>] branch created by user1", pl.Text)
	})

	t.Run("Delete", func(t *testing.T) {
		p := deleteTestPayload()

		pl, err := sc.Delete(p)
		require.NoError(t, err)

		assert.Equal(t, "[<http://localhost:3000/test/repo|test/repo>:test] branch deleted by user1", pl.Text)
	})

	t.Run("Fork", func(t *testing.T) {
		p := forkTestPayload()

		pl, err := sc.Fork(p)
		require.NoError(t, err)

		assert.Equal(t, "<http://localhost:3000/test/repo2|test/repo2> is forked to <http://localhost:3000/test/repo|test/repo>", pl.Text)
	})

	t.Run("Push", func(t *testing.T) {
		p := pushTestPayload()

		pl, err := sc.Push(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo:<http://localhost:3000/test/repo/src/branch/test|test>] 2 new commits pushed by user1", pl.Text)
	})

	t.Run("Issue", func(t *testing.T) {
		p := issueTestPayload()

		p.Action = api.HookIssueOpened
		pl, err := sc.Issue(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] Issue opened: <http://localhost:3000/test/repo/issues/2|#2 crash> by user1", pl.Text)

		p.Action = api.HookIssueClosed
		pl, err = sc.Issue(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] Issue closed: <http://localhost:3000/test/repo/issues/2|#2 crash> by user1", pl.Text)
	})

	t.Run("IssueComment", func(t *testing.T) {
		p := issueCommentTestPayload()

		pl, err := sc.IssueComment(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] New comment on issue <http://localhost:3000/test/repo/issues/2|#2 crash> by user1", pl.Text)
	})

	t.Run("PullRequest", func(t *testing.T) {
		p := pullRequestTestPayload()

		pl, err := sc.PullRequest(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] Pull request opened: <http://localhost:3000/test/repo/pulls/12|#12 Fix bug> by user1", pl.Text)
	})

	t.Run("PullRequestComment", func(t *testing.T) {
		p := pullRequestCommentTestPayload()

		pl, err := sc.IssueComment(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] New comment on pull request <http://localhost:3000/test/repo/pulls/12|#12 Fix bug> by user1", pl.Text)
	})

	t.Run("Review", func(t *testing.T) {
		p := pullRequestTestPayload()
		p.Action = api.HookIssueReviewed

		pl, err := sc.Review(p, webhook_module.HookEventPullRequestReviewApproved)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] Pull request review approved: [#12 Fix bug](http://localhost:3000/test/repo/pulls/12) by user1", pl.Text)
	})

	t.Run("Repository", func(t *testing.T) {
		p := repositoryTestPayload()

		pl, err := sc.Repository(p)
		require.NoError(t, err)

		assert.Equal(t, "[<http://localhost:3000/test/repo|test/repo>] Repository created by user1", pl.Text)
	})

	t.Run("Package", func(t *testing.T) {
		p := packageTestPayload()

		pl, err := sc.Package(p)
		require.NoError(t, err)

		assert.Equal(t, "Package created: <http://localhost:3000/user1/-/packages/container/GiteaContainer/latest|GiteaContainer:latest> by user1", pl.Text)
	})

	t.Run("Wiki", func(t *testing.T) {
		p := wikiTestPayload()

		p.Action = api.HookWikiCreated
		pl, err := sc.Wiki(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] New wiki page '<http://localhost:3000/test/repo/wiki/index|index>' (Wiki change comment) by user1", pl.Text)

		p.Action = api.HookWikiEdited
		pl, err = sc.Wiki(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] Wiki page '<http://localhost:3000/test/repo/wiki/index|index>' edited (Wiki change comment) by user1", pl.Text)

		p.Action = api.HookWikiDeleted
		pl, err = sc.Wiki(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] Wiki page '<http://localhost:3000/test/repo/wiki/index|index>' deleted by user1", pl.Text)
	})

	t.Run("Release", func(t *testing.T) {
		p := pullReleaseTestPayload()

		pl, err := sc.Release(p)
		require.NoError(t, err)

		assert.Equal(t, "[test/repo] Release created: <http://localhost:3000/test/repo/releases/tag/v1.0|v1.0> by user1", pl.Text)
	})
}

func TestSlackJSONPayload(t *testing.T) {
	p := pushTestPayload()
	data, err := p.JSONPayload()
	require.NoError(t, err)

	hook := &webhook_model.Webhook{
		RepoID:     3,
		IsActive:   true,
		Type:       webhook_module.SLACK,
		URL:        "https://slack.example.com/",
		Meta:       `{}`,
		HTTPMethod: "POST",
	}
	task := &webhook_model.HookTask{
		HookID:         hook.ID,
		EventType:      webhook_module.HookEventPush,
		PayloadContent: string(data),
		PayloadVersion: 2,
	}

	req, reqBody, err := slackHandler{}.NewRequest(t.Context(), hook, task)
	require.NotNil(t, req)
	require.NotNil(t, reqBody)
	require.NoError(t, err)

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "https://slack.example.com/", req.URL.String())
	assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	var body SlackPayload
	err = json.NewDecoder(req.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "[test/repo:<http://localhost:3000/test/repo/src/branch/test|test>] 2 new commits pushed by user1", body.Text)
}

func TestIsValidSlackChannel(t *testing.T) {
	tt := []struct {
		channelName string
		expected    bool
	}{
		{"gitea", true},
		{"#gitea", true},
		{"  ", false},
		{"#", false},
		{" #", false},
		{"gitea   ", false},
		{"  gitea", false},
	}

	for _, v := range tt {
		assert.Equal(t, v.expected, IsValidSlackChannel(v.channelName))
	}
}

func TestSlackMetadata(t *testing.T) {
	w := &webhook_model.Webhook{
		Meta: `{"channel": "foo", "username": "username", "color": "blue"}`,
	}
	slackHook := slackHandler{}.Metadata(w)
	assert.Equal(t, SlackMeta{
		Channel:  "foo",
		Username: "username",
		Color:    "blue",
	},
		*slackHook.(*SlackMeta))
}

func TestSlackToHook(t *testing.T) {
	w := &webhook_model.Webhook{
		Type:        webhook_module.SLACK,
		ContentType: webhook_model.ContentTypeJSON,
		URL:         "https://slack.example.com",
		Meta:        `{"channel": "foo", "username": "username", "color": "blue"}`,
		HookEvent: &webhook_module.HookEvent{
			PushOnly:       true,
			SendEverything: false,
			ChooseEvents:   false,
			HookEvents: webhook_module.HookEvents{
				Create:      false,
				Push:        true,
				PullRequest: false,
			},
		},
	}
	h, err := ToHook("repoLink", w)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"url":          "https://slack.example.com",
		"content_type": "json",

		"channel":  "foo",
		"color":    "blue",
		"icon_url": "",
		"username": "username",
	}, h.Config)
	assert.Equal(t, "https://slack.example.com", h.URL)
	assert.Equal(t, "json", h.ContentType)
	assert.Equal(t, &SlackMeta{
		Channel:  "foo",
		Username: "username",
		IconURL:  "",
		Color:    "blue",
	}, h.Metadata)
}
