// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
)

// doMergeStyleMerge merges the tracking branch into the current HEAD - which is assumed to be the staging branch (equal to the pr.BaseBranch)
func doMergeStyleMerge(ctx *mergeContext, message string) error {
	cmd := git.NewCommand(ctx, "merge", "--no-ff", "--no-commit").AddDynamicArguments(trackingBranch)
	if err := runMergeCommand(ctx, repo_model.MergeStyleMerge, cmd); err != nil {
		log.Error("%-v Unable to merge tracking into base: %v", ctx.pr, err)
		return err
	}

	if err := commitAndSignNoAuthor(ctx, message); err != nil {
		log.Error("%-v Unable to make final commit: %v", ctx.pr, err)
		return err
	}
	return nil
}
