// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	notify_service "forgejo.org/services/notify"
)

// GenerateIssueLabels generates issue labels from a template repository
func GenerateIssueLabels(ctx context.Context, templateRepo, generateRepo *repo_model.Repository) error {
	templateLabels, err := issues_model.GetLabelsByRepoID(ctx, templateRepo.ID, "", db.ListOptions{})
	if err != nil {
		return err
	}
	// Prevent insert being called with an empty slice which would result in
	// err "no element on slice when insert".
	if len(templateLabels) == 0 {
		return nil
	}

	newLabels := make([]*issues_model.Label, 0, len(templateLabels))
	for _, templateLabel := range templateLabels {
		newLabels = append(newLabels, &issues_model.Label{
			RepoID:      generateRepo.ID,
			Name:        templateLabel.Name,
			Exclusive:   templateLabel.Exclusive,
			Description: templateLabel.Description,
			Color:       templateLabel.Color,
		})
	}
	return db.Insert(ctx, newLabels)
}

func GenerateProtectedBranch(ctx context.Context, templateRepo, generateRepo *repo_model.Repository) error {
	templateBranches, err := git_model.FindRepoProtectedBranchRules(ctx, templateRepo.ID)
	if err != nil {
		return err
	}
	// Prevent insert being called with an empty slice which would result in
	// err "no element on slice when insert".
	if len(templateBranches) == 0 {
		return nil
	}

	newBranches := make([]*git_model.ProtectedBranch, 0, len(templateBranches))
	for _, templateBranch := range templateBranches {
		templateBranch.ID = 0
		templateBranch.RepoID = generateRepo.ID
		templateBranch.UpdatedUnix = 0
		templateBranch.CreatedUnix = 0
		newBranches = append(newBranches, templateBranch)
	}
	return db.Insert(ctx, newBranches)
}

// GenerateRepository generates a repository from a template
func GenerateRepository(ctx context.Context, doer, owner *user_model.User, templateRepo *repo_model.Repository, opts GenerateRepoOptions) (_ *repo_model.Repository, err error) {
	if !doer.IsAdmin && !owner.CanCreateRepo() {
		return nil, repo_model.ErrReachLimitOfRepo{
			Limit: owner.MaxRepoCreation,
		}
	}

	var generateRepo *repo_model.Repository
	if err = db.WithTx(ctx, func(ctx context.Context) error {
		generateRepo, err = generateRepository(ctx, doer, owner, templateRepo, opts)
		if err != nil {
			return err
		}

		// Git Content
		if opts.GitContent && !templateRepo.IsEmpty {
			if err = GenerateGitContent(ctx, templateRepo, generateRepo); err != nil {
				return err
			}
		}

		// Topics
		if opts.Topics {
			if err = repo_model.GenerateTopics(ctx, templateRepo, generateRepo); err != nil {
				return err
			}
		}

		// Git Hooks
		if opts.GitHooks {
			if err = GenerateGitHooks(ctx, templateRepo, generateRepo); err != nil {
				return err
			}
		}

		// Webhooks
		if opts.Webhooks {
			if err = GenerateWebhooks(ctx, templateRepo, generateRepo); err != nil {
				return err
			}
		}

		// Avatar
		if opts.Avatar && len(templateRepo.Avatar) > 0 {
			if err = generateAvatar(ctx, templateRepo, generateRepo); err != nil {
				return err
			}
		}

		// Issue Labels
		if opts.IssueLabels {
			if err = GenerateIssueLabels(ctx, templateRepo, generateRepo); err != nil {
				return err
			}
		}

		if opts.ProtectedBranch {
			if err = GenerateProtectedBranch(ctx, templateRepo, generateRepo); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	notify_service.CreateRepository(ctx, doer, owner, generateRepo)

	return generateRepo, nil
}
