// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitrepo

import (
	"context"

	"forgejo.org/modules/git"
)

// WalkReferences walks all the references from the repository
func WalkReferences(ctx context.Context, repo Repository, walkfn func(sha1, refname string) error) (int, error) {
	return git.WalkShowRef(ctx, repoPath(repo), nil, 0, 0, walkfn)
}
