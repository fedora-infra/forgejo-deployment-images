// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withKeyFile(t *testing.T, keyname string, callback func(string)) {
	tmpDir := t.TempDir()

	err := os.Chmod(tmpDir, 0o700)
	require.NoError(t, err)

	keyFile := filepath.Join(tmpDir, keyname)
	pubkey, privkey, err := util.GenerateSSHKeypair()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(keyFile, privkey, 0o600))
	require.NoError(t, os.WriteFile(keyFile+".pub", pubkey, 0o600))

	err = os.WriteFile(path.Join(tmpDir, "ssh"), []byte("#!/bin/bash\n"+
		"ssh -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -o \"IdentitiesOnly=yes\" -i \""+keyFile+"\" \"$@\""), 0o700)
	require.NoError(t, err)

	// Setup ssh wrapper
	t.Setenv("GIT_SSH", path.Join(tmpDir, "ssh"))
	t.Setenv("GIT_SSH_COMMAND",
		"ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o IdentitiesOnly=yes -i \""+keyFile+"\"")
	t.Setenv("GIT_SSH_VARIANT", "ssh")

	callback(keyFile)
}

func createSSHUrl(gitPath string, u *url.URL) *url.URL {
	u2 := *u
	u2.Scheme = "ssh"
	u2.User = url.User("git")
	u2.Host = net.JoinHostPort(setting.SSH.ListenHost, strconv.Itoa(setting.SSH.ListenPort))
	u2.Path = gitPath
	return &u2
}

var rootPathRe = regexp.MustCompile("\\[repository\\]\nROOT\\s=\\s.*")

func onGiteaRun[T testing.TB](t T, callback func(T, *url.URL)) {
	defer tests.PrepareTestEnv(t, 1)()
	s := http.Server{
		Handler: testWebRoutes,
	}

	u, err := url.Parse(setting.AppURL)
	require.NoError(t, err)
	listener, err := net.Listen("tcp", u.Host)
	i := 0
	for err != nil && i <= 10 {
		time.Sleep(100 * time.Millisecond)
		listener, err = net.Listen("tcp", u.Host)
		i++
	}
	require.NoError(t, err)
	u.Host = listener.Addr().String()

	// Override repository root in config.
	conf, err := os.ReadFile(setting.CustomConf)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(setting.CustomConf, rootPathRe.ReplaceAll(conf, []byte("[repository]\nROOT = "+setting.RepoRootPath)), 0o600))

	defer func() {
		require.NoError(t, os.WriteFile(setting.CustomConf, conf, 0o600))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		s.Shutdown(ctx)
		cancel()
	}()

	go s.Serve(listener)
	// Started by config go ssh.Listen(setting.SSH.ListenHost, setting.SSH.ListenPort, setting.SSH.ServerCiphers, setting.SSH.ServerKeyExchanges, setting.SSH.ServerMACs)

	callback(t, u)
}

func doGitClone(dstLocalPath string, u *url.URL) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		require.NoError(t, git.CloneWithArgs(t.Context(), git.AllowLFSFiltersArgs(), u.String(), dstLocalPath, git.CloneRepoOptions{}))
		exist, err := util.IsExist(filepath.Join(dstLocalPath, "README.md"))
		require.NoError(t, err)
		assert.True(t, exist)
	}
}

func doPartialGitClone(dstLocalPath string, u *url.URL) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		require.NoError(t, git.CloneWithArgs(t.Context(), git.AllowLFSFiltersArgs(), u.String(), dstLocalPath, git.CloneRepoOptions{
			Filter: "blob:none",
		}))
		exist, err := util.IsExist(filepath.Join(dstLocalPath, "README.md"))
		require.NoError(t, err)
		assert.True(t, exist)
	}
}

func doGitCloneFail(u *url.URL) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		tmpDir := t.TempDir()
		require.Error(t, git.Clone(git.DefaultContext, u.String(), tmpDir, git.CloneRepoOptions{}))
		exist, err := util.IsExist(filepath.Join(tmpDir, "README.md"))
		require.NoError(t, err)
		assert.False(t, exist)
	}
}

func doGitInitTestRepository(dstPath string, objectFormat git.ObjectFormat) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		// Init repository in dstPath
		require.NoError(t, git.InitRepository(git.DefaultContext, dstPath, false, objectFormat.Name()))
		// forcibly set default branch to master
		_, _, err := git.NewCommand(git.DefaultContext, "symbolic-ref", "HEAD", git.BranchPrefix+"master").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dstPath, "README.md"), []byte(fmt.Sprintf("# Testing Repository\n\nOriginally created in: %s", dstPath)), 0o644))
		require.NoError(t, git.AddChanges(dstPath, true))
		signature := git.Signature{
			Email: "test@example.com",
			Name:  "test",
			When:  time.Now(),
		}
		require.NoError(t, git.CommitChanges(dstPath, git.CommitChangesOptions{
			Committer: &signature,
			Author:    &signature,
			Message:   "Initial Commit",
		}))
	}
}

func doGitAddRemote(dstPath, remoteName string, u *url.URL) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		_, _, err := git.NewCommand(git.DefaultContext, "remote", "add").AddDynamicArguments(remoteName, u.String()).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
	}
}

func doGitPushTestRepository(dstPath string, args ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		_, _, err := git.NewCommand(git.DefaultContext, "push", "-u").AddArguments(git.ToTrustedCmdArgs(args)...).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
	}
}

func doGitPushTestRepositoryFail(dstPath string, args ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		_, _, err := git.NewCommand(git.DefaultContext, "push").AddArguments(git.ToTrustedCmdArgs(args)...).RunStdString(&git.RunOpts{Dir: dstPath})
		require.Error(t, err)
	}
}

func doGitAddSomeCommits(dstPath, branch string) func(*testing.T) {
	return func(t *testing.T) {
		doGitCheckoutBranch(dstPath, branch)(t)

		require.NoError(t, os.WriteFile(filepath.Join(dstPath, fmt.Sprintf("file-%s.txt", branch)), []byte(fmt.Sprintf("file %s", branch)), 0o644))
		require.NoError(t, git.AddChanges(dstPath, true))
		signature := git.Signature{
			Email: "test@test.test",
			Name:  "test",
		}
		require.NoError(t, git.CommitChanges(dstPath, git.CommitChangesOptions{
			Committer: &signature,
			Author:    &signature,
			Message:   fmt.Sprintf("update %s", branch),
		}))
	}
}

func doGitCreateBranch(dstPath, branch string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		_, _, err := git.NewCommand(git.DefaultContext, "checkout", "-b").AddDynamicArguments(branch).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
	}
}

func doGitCheckoutBranch(dstPath string, args ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		_, _, err := git.NewCommandContextNoGlobals(git.DefaultContext, git.AllowLFSFiltersArgs()...).AddArguments("checkout").AddArguments(git.ToTrustedCmdArgs(args)...).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
	}
}

func doGitPull(dstPath string, args ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		_, _, err := git.NewCommandContextNoGlobals(git.DefaultContext, git.AllowLFSFiltersArgs()...).AddArguments("pull").AddArguments(git.ToTrustedCmdArgs(args)...).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
	}
}
