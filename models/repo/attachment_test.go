// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncreaseDownloadCount(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	attachment, err := repo_model.GetAttachmentByUUID(db.DefaultContext, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	require.NoError(t, err)
	assert.Equal(t, int64(0), attachment.DownloadCount)

	// increase download count
	err = attachment.IncreaseDownloadCount(db.DefaultContext)
	require.NoError(t, err)

	attachment, err = repo_model.GetAttachmentByUUID(db.DefaultContext, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	require.NoError(t, err)
	assert.Equal(t, int64(1), attachment.DownloadCount)
}

func TestGetByCommentOrIssueID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// count of attachments from issue ID
	attachments, err := repo_model.GetAttachmentsByIssueID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, attachments, 1)

	attachments, err = repo_model.GetAttachmentsByCommentID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, attachments, 2)
}

func TestDeleteAttachments(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	count, err := repo_model.DeleteAttachmentsByComment(db.DefaultContext, 2, false)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	err = repo_model.DeleteAttachment(db.DefaultContext, &repo_model.Attachment{ID: 8}, false)
	require.NoError(t, err)

	attachment, err := repo_model.GetAttachmentByUUID(db.DefaultContext, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a18")
	require.Error(t, err)
	assert.True(t, repo_model.IsErrAttachmentNotExist(err))
	assert.Nil(t, attachment)
}

func TestGetAttachmentByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	attach, err := repo_model.GetAttachmentByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", attach.UUID)
}

func TestAttachment_DownloadURL(t *testing.T) {
	attach := &repo_model.Attachment{
		UUID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		ID:   1,
	}
	assert.Equal(t, "https://try.gitea.io/attachments/a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", attach.DownloadURL())
}

func TestUpdateAttachment(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	attach, err := repo_model.GetAttachmentByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", attach.UUID)

	attach.Name = "new_name"
	require.NoError(t, repo_model.UpdateAttachment(db.DefaultContext, attach))

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{Name: "new_name"})
}

func TestGetAttachmentsByUUIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	attachList, err := repo_model.GetAttachmentsByUUIDs(db.DefaultContext, []string{"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a17", "not-existing-uuid"})
	require.NoError(t, err)
	assert.Len(t, attachList, 2)
	assert.Equal(t, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", attachList[0].UUID)
	assert.Equal(t, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a17", attachList[1].UUID)
	assert.Equal(t, int64(1), attachList[0].IssueID)
	assert.Equal(t, int64(5), attachList[1].IssueID)
}
