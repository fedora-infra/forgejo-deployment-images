// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	repo_model "forgejo.org/models/repo"
	api "forgejo.org/modules/structs"
)

// ToPushMirror convert from repo_model.PushMirror and remoteAddress to api.TopicResponse
func ToPushMirror(ctx context.Context, pm *repo_model.PushMirror) (*api.PushMirror, error) {
	repo := pm.GetRepository(ctx)
	return &api.PushMirror{
		RepoName:       repo.Name,
		RemoteName:     pm.RemoteName,
		RemoteAddress:  pm.RemoteAddress,
		CreatedUnix:    pm.CreatedUnix.AsTime(),
		LastUpdateUnix: pm.LastUpdateUnix.AsTimePtr(),
		LastError:      pm.LastError,
		Interval:       pm.Interval.String(),
		SyncOnCommit:   pm.SyncOnCommit,
		PublicKey:      pm.GetPublicKey(),
	}, nil
}
