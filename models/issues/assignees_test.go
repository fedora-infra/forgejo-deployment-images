// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAssignee(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Fake issue with assignees
	issue, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	require.NoError(t, err)

	err = issue.LoadAttributes(db.DefaultContext)
	require.NoError(t, err)

	// Assign multiple users
	user2, err := user_model.GetUserByID(db.DefaultContext, 2)
	require.NoError(t, err)
	_, _, err = issues_model.ToggleIssueAssignee(db.DefaultContext, issue, &user_model.User{ID: 1}, user2.ID)
	require.NoError(t, err)

	org3, err := user_model.GetUserByID(db.DefaultContext, 3)
	require.NoError(t, err)
	_, _, err = issues_model.ToggleIssueAssignee(db.DefaultContext, issue, &user_model.User{ID: 1}, org3.ID)
	require.NoError(t, err)

	user1, err := user_model.GetUserByID(db.DefaultContext, 1) // This user is already assigned (see the definition in fixtures), so running  UpdateAssignee should unassign him
	require.NoError(t, err)
	_, _, err = issues_model.ToggleIssueAssignee(db.DefaultContext, issue, &user_model.User{ID: 1}, user1.ID)
	require.NoError(t, err)

	// Check if he got removed
	isAssigned, err := issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, user1)
	require.NoError(t, err)
	assert.False(t, isAssigned)

	// Check if they're all there
	err = issue.LoadAssignees(db.DefaultContext)
	require.NoError(t, err)

	var expectedAssignees []*user_model.User
	expectedAssignees = append(expectedAssignees, user2, org3)

	for in, assignee := range issue.Assignees {
		assert.Equal(t, assignee.ID, expectedAssignees[in].ID)
	}

	// Check if the user is assigned
	isAssigned, err = issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, user2)
	require.NoError(t, err)
	assert.True(t, isAssigned)

	// This user should not be assigned
	isAssigned, err = issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, &user_model.User{ID: 4})
	require.NoError(t, err)
	assert.False(t, isAssigned)
}

func TestMakeIDsFromAPIAssigneesToAdd(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	_ = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	_ = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	IDs, err := issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "", []string{""})
	require.NoError(t, err)
	assert.Equal(t, []int64{}, IDs)

	_, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "", []string{"none_existing_user"})
	require.Error(t, err)

	IDs, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "user1", []string{"user1"})
	require.NoError(t, err)
	assert.Equal(t, []int64{1}, IDs)

	IDs, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "user2", []string{""})
	require.NoError(t, err)
	assert.Equal(t, []int64{2}, IDs)

	IDs, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "", []string{"user1", "user2"})
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, IDs)
}
