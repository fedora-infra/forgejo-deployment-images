// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/lfs"
	"forgejo.org/modules/log"
	"forgejo.org/modules/migration"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// MigrateRepositoryGitData starts migrating git related data after created migrating repository
func MigrateRepositoryGitData(ctx context.Context, u *user_model.User,
	repo *repo_model.Repository, opts migration.MigrateOptions,
	httpTransport *http.Transport,
) (*repo_model.Repository, error) {
	repoPath := repo_model.RepoPath(u.Name, opts.RepoName)

	if u.IsOrganization() {
		t, err := organization.OrgFromUser(u).GetOwnerTeam(ctx)
		if err != nil {
			return nil, err
		}
		repo.NumWatches = t.NumMembers
	} else {
		repo.NumWatches = 1
	}

	migrateTimeout := time.Duration(setting.Git.Timeout.Migrate) * time.Second

	var err error
	if err = util.RemoveAll(repoPath); err != nil {
		return repo, fmt.Errorf("Failed to remove %s: %w", repoPath, err)
	}

	if err = git.Clone(ctx, opts.CloneAddr, repoPath, git.CloneRepoOptions{
		Mirror:        true,
		Quiet:         true,
		Timeout:       migrateTimeout,
		SkipTLSVerify: setting.Migrations.SkipTLSVerify,
	}); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return repo, fmt.Errorf("Clone timed out. Consider increasing [git.timeout] MIGRATE in app.ini. Underlying Error: %w", err)
		}
		return repo, fmt.Errorf("Clone: %w", err)
	}

	if err := git.WriteCommitGraph(ctx, repoPath); err != nil {
		return repo, err
	}

	if opts.Wiki {
		wikiPath := repo_model.WikiPath(u.Name, opts.RepoName)
		wikiRemotePath := repo_module.WikiRemoteURL(ctx, opts.CloneAddr)
		if len(wikiRemotePath) > 0 {
			if err := util.RemoveAll(wikiPath); err != nil {
				return repo, fmt.Errorf("Failed to remove %s: %w", wikiPath, err)
			}

			if err := git.Clone(ctx, wikiRemotePath, wikiPath, git.CloneRepoOptions{
				Mirror:        true,
				Quiet:         true,
				Timeout:       migrateTimeout,
				SkipTLSVerify: setting.Migrations.SkipTLSVerify,
			}); err != nil {
				log.Warn("Clone wiki: %v", err)
				if err := util.RemoveAll(wikiPath); err != nil {
					return repo, fmt.Errorf("Failed to remove %s: %w", wikiPath, err)
				}
			} else {
				// Figure out the branch of the wiki we just cloned. We assume
				// that the default branch is to be used, and we'll use the same
				// name as the source.
				gitRepo, err := git.OpenRepository(ctx, wikiPath)
				if err != nil {
					log.Warn("Failed to open wiki repository during migration: %v", err)
					if err := util.RemoveAll(wikiPath); err != nil {
						return repo, fmt.Errorf("Failed to remove %s: %w", wikiPath, err)
					}
					return repo, err
				}
				defer gitRepo.Close()

				branch, err := gitRepo.GetHEADBranch()
				if err != nil {
					log.Warn("Failed to get the default branch of a migrated wiki repo: %v", err)
					if err := util.RemoveAll(wikiPath); err != nil {
						return repo, fmt.Errorf("Failed to remove %s: %w", wikiPath, err)
					}

					return repo, err
				}
				repo.WikiBranch = branch.Name

				if err := git.WriteCommitGraph(ctx, wikiPath); err != nil {
					return repo, err
				}
			}
		}
	}

	if repo.OwnerID == u.ID {
		repo.Owner = u
	}

	if err = repo_module.CheckDaemonExportOK(ctx, repo); err != nil {
		return repo, fmt.Errorf("checkDaemonExportOK: %w", err)
	}

	if stdout, _, err := git.NewCommand(ctx, "update-server-info").
		SetDescription(fmt.Sprintf("MigrateRepositoryGitData(git update-server-info): %s", repoPath)).
		RunStdString(&git.RunOpts{Dir: repoPath}); err != nil {
		log.Error("MigrateRepositoryGitData(git update-server-info) in %v: Stdout: %s\nError: %v", repo, stdout, err)
		return repo, fmt.Errorf("error in MigrateRepositoryGitData(git update-server-info): %w", err)
	}

	gitRepo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		return repo, fmt.Errorf("OpenRepository: %w", err)
	}
	defer gitRepo.Close()

	repo.IsEmpty, err = gitRepo.IsEmpty()
	if err != nil {
		return repo, fmt.Errorf("git.IsEmpty: %w", err)
	}

	if !repo.IsEmpty {
		if len(repo.DefaultBranch) == 0 {
			// Try to get HEAD branch and set it as default branch.
			headBranch, err := gitRepo.GetHEADBranch()
			if err != nil {
				return repo, fmt.Errorf("GetHEADBranch: %w", err)
			}
			if headBranch != nil {
				repo.DefaultBranch = headBranch.Name
			}
		}

		if _, err := repo_module.SyncRepoBranchesWithRepo(ctx, repo, gitRepo, u.ID); err != nil {
			return repo, fmt.Errorf("SyncRepoBranchesWithRepo: %v", err)
		}

		if !opts.Releases {
			// note: this will greatly improve release (tag) sync
			// for pull-mirrors with many tags
			repo.IsMirror = opts.Mirror
			if err = repo_module.SyncReleasesWithTags(ctx, repo, gitRepo); err != nil {
				log.Error("Failed to synchronize tags to releases for repository: %v", err)
			}
		}

		if opts.LFS {
			endpoint := lfs.DetermineEndpoint(opts.CloneAddr, opts.LFSEndpoint)
			lfsClient := lfs.NewClient(endpoint, httpTransport)
			if err = repo_module.StoreMissingLfsObjectsInRepository(ctx, repo, gitRepo, lfsClient); err != nil {
				log.Error("Failed to store missing LFS objects for repository: %v", err)
				return repo, fmt.Errorf("StoreMissingLfsObjectsInRepository: %w", err)
			}
		}
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return nil, err
	}
	defer committer.Close()

	if opts.Mirror {
		remoteAddress, err := util.SanitizeURL(opts.CloneAddr)
		if err != nil {
			return repo, err
		}
		mirrorModel := repo_model.Mirror{
			RepoID:         repo.ID,
			Interval:       setting.Mirror.DefaultInterval,
			EnablePrune:    true,
			NextUpdateUnix: timeutil.TimeStampNow().AddDuration(setting.Mirror.DefaultInterval),
			LFS:            opts.LFS,
			RemoteAddress:  remoteAddress,
		}
		if opts.LFS {
			mirrorModel.LFSEndpoint = opts.LFSEndpoint
		}

		if opts.MirrorInterval != "" {
			parsedInterval, err := time.ParseDuration(opts.MirrorInterval)
			if err != nil {
				log.Error("Failed to set Interval: %v", err)
				return repo, err
			}
			if parsedInterval == 0 {
				mirrorModel.Interval = 0
				mirrorModel.NextUpdateUnix = 0
			} else if parsedInterval < setting.Mirror.MinInterval {
				err := fmt.Errorf("interval %s is set below Minimum Interval of %s", parsedInterval, setting.Mirror.MinInterval)
				log.Error("Interval: %s is too frequent", opts.MirrorInterval)
				return repo, err
			} else {
				mirrorModel.Interval = parsedInterval
				mirrorModel.NextUpdateUnix = timeutil.TimeStampNow().AddDuration(parsedInterval)
			}
		}

		if err = repo_model.InsertMirror(ctx, &mirrorModel); err != nil {
			return repo, fmt.Errorf("InsertOne: %w", err)
		}

		repo.IsMirror = true
		if err = UpdateRepository(ctx, repo, false); err != nil {
			return nil, err
		}

		// this is necessary for sync local tags from remote
		configName := fmt.Sprintf("remote.%s.fetch", mirrorModel.GetRemoteName())
		if stdout, _, err := git.NewCommand(ctx, "config").
			AddOptionValues("--add", configName, `+refs/tags/*:refs/tags/*`).
			RunStdString(&git.RunOpts{Dir: repoPath}); err != nil {
			log.Error("MigrateRepositoryGitData(git config --add <remote> +refs/tags/*:refs/tags/*) in %v: Stdout: %s\nError: %v", repo, stdout, err)
			return repo, fmt.Errorf("error in MigrateRepositoryGitData(git config --add <remote> +refs/tags/*:refs/tags/*): %w", err)
		}
	} else {
		if err = repo_module.UpdateRepoSize(ctx, repo); err != nil {
			log.Error("Failed to update size for repository: %v", err)
		}
		if repo, err = CleanUpMigrateInfo(ctx, repo); err != nil {
			return nil, err
		}
	}

	return repo, committer.Commit()
}

// cleanUpMigrateGitConfig removes mirror info which prevents "push --all".
// This also removes possible user credentials.
func cleanUpMigrateGitConfig(ctx context.Context, repoPath string) error {
	cmd := git.NewCommand(ctx, "remote", "rm", "origin")
	// if the origin does not exist
	_, _, err := cmd.RunStdString(&git.RunOpts{
		Dir: repoPath,
	})
	if err != nil && !git.IsRemoteNotExistError(err) {
		return err
	}
	return nil
}

// CleanUpMigrateInfo finishes migrating repository and/or wiki with things that don't need to be done for mirrors.
func CleanUpMigrateInfo(ctx context.Context, repo *repo_model.Repository) (*repo_model.Repository, error) {
	repoPath := repo.RepoPath()
	if err := repo_module.CreateDelegateHooks(repoPath); err != nil {
		return repo, fmt.Errorf("createDelegateHooks: %w", err)
	}
	if repo.HasWiki() {
		if err := repo_module.CreateDelegateHooks(repo.WikiPath()); err != nil {
			return repo, fmt.Errorf("createDelegateHooks.(wiki): %w", err)
		}
	}

	_, _, err := git.NewCommand(ctx, "remote", "rm", "origin").RunStdString(&git.RunOpts{Dir: repoPath})
	if err != nil && !git.IsRemoteNotExistError(err) {
		return repo, fmt.Errorf("CleanUpMigrateInfo: %w", err)
	}

	if repo.HasWiki() {
		if err := cleanUpMigrateGitConfig(ctx, repo.WikiPath()); err != nil {
			return repo, fmt.Errorf("cleanUpMigrateGitConfig (wiki): %w", err)
		}
	}

	return repo, UpdateRepository(ctx, repo, false)
}
