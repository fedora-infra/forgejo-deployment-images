// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	activities_model "forgejo.org/models/activities"
	perm_model "forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	api "forgejo.org/modules/structs"
)

func ToActivity(ctx context.Context, ac *activities_model.Action, doer *user_model.User) *api.Activity {
	p, err := access_model.GetUserRepoPermission(ctx, ac.Repo, doer)
	if err != nil {
		log.Error("GetUserRepoPermission[%d]: %v", ac.RepoID, err)
		p.AccessMode = perm_model.AccessModeNone
	}

	result := &api.Activity{
		ID:        ac.ID,
		UserID:    ac.UserID,
		OpType:    ac.OpType.String(),
		ActUserID: ac.ActUserID,
		ActUser:   ToUser(ctx, ac.ActUser, doer),
		RepoID:    ac.RepoID,
		Repo:      ToRepo(ctx, ac.Repo, p),
		RefName:   ac.RefName,
		IsPrivate: ac.IsPrivate,
		Content:   ac.Content,
		Created:   ac.CreatedUnix.AsTime(),
	}

	if ac.Comment != nil {
		result.CommentID = ac.CommentID
		result.Comment = ToAPIComment(ctx, ac.Repo, ac.Comment)
	}

	return result
}

func ToActivities(ctx context.Context, al activities_model.ActionList, doer *user_model.User) []*api.Activity {
	result := make([]*api.Activity, 0, len(al))
	for _, ac := range al {
		result = append(result, ToActivity(ctx, ac, doer))
	}
	return result
}
