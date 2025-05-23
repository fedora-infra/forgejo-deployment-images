// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization

import (
	"context"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
)

// GetOrgRepositories get repos belonging to the given organization
func GetOrgRepositories(ctx context.Context, orgID int64) (repo_model.RepositoryList, error) {
	var orgRepos []*repo_model.Repository
	return orgRepos, db.GetEngine(ctx).Where("owner_id = ?", orgID).Find(&orgRepos)
}
