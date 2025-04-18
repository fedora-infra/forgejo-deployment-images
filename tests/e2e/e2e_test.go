// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// This is primarily coped from /tests/integration/integration_test.go
//   TODO: Move common functions to shared file

//nolint:forbidigo
package e2e

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/testlogger"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers"
	"code.gitea.io/gitea/tests"
)

var testE2eWebRoutes *web.Route

func TestMain(m *testing.M) {
	defer log.GetManager().Close()

	managerCtx, cancel := context.WithCancel(context.Background())
	graceful.InitManager(managerCtx)
	defer cancel()

	tests.InitTest(true)
	initChangedFiles()
	testE2eWebRoutes = routers.NormalRoutes()

	os.Unsetenv("GIT_AUTHOR_NAME")
	os.Unsetenv("GIT_AUTHOR_EMAIL")
	os.Unsetenv("GIT_AUTHOR_DATE")
	os.Unsetenv("GIT_COMMITTER_NAME")
	os.Unsetenv("GIT_COMMITTER_EMAIL")
	os.Unsetenv("GIT_COMMITTER_DATE")

	err := unittest.InitFixtures(
		unittest.FixturesOptions{
			Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
			Base: setting.AppWorkPath,
			Dirs: []string{"tests/e2e/fixtures/"},
		},
	)
	if err != nil {
		fmt.Printf("Error initializing test database: %v\n", err)
		os.Exit(1)
	}

	exitVal := m.Run()

	if err := testlogger.WriterCloser.Reset(); err != nil {
		fmt.Printf("testlogger.WriterCloser.Reset: error ignored: %v\n", err)
	}
	if err = util.RemoveAll(setting.Indexer.IssuePath); err != nil {
		fmt.Printf("util.RemoveAll: %v\n", err)
		os.Exit(1)
	}
	if err = util.RemoveAll(setting.Indexer.RepoPath); err != nil {
		fmt.Printf("Unable to remove repo indexer: %v\n", err)
		os.Exit(1)
	}

	os.Exit(exitVal)
}

// TestE2e should be the only test e2e necessary. It will collect all "*.test.e2e.js" files in this directory and build a test for each.
func TestE2e(t *testing.T) {
	// Find the paths of all e2e test files in test directory.
	searchGlob := filepath.Join(filepath.Dir(setting.AppPath), "tests", "e2e", "*.test.e2e.ts")
	paths, err := filepath.Glob(searchGlob)
	if err != nil {
		t.Fatal(err)
	} else if len(paths) == 0 {
		t.Fatal(fmt.Errorf("No e2e tests found in %s", searchGlob))
	}

	runArgs := []string{"npx", "playwright", "test"}

	_, testVisual := os.LookupEnv("VISUAL_TEST")
	// To update snapshot outputs
	if _, set := os.LookupEnv("ACCEPT_VISUAL"); set {
		runArgs = append(runArgs, "--update-snapshots")
	}
	if project := os.Getenv("PLAYWRIGHT_PROJECT"); project != "" {
		runArgs = append(runArgs, "--project="+project)
	}

	// Create new test for each input file
	for _, path := range paths {
		_, filename := filepath.Split(path)
		testname := filename[:len(filename)-len(filepath.Ext(path))]

		if canSkipTest(path) {
			fmt.Printf("No related changes for test, skipping: %s\n", filename)
			continue
		}

		t.Run(testname, func(t *testing.T) {
			// Default 2 minute timeout
			onForgejoRun(t, func(*testing.T, *url.URL) {
				defer DeclareGitRepos(t)()
				thisTest := runArgs
				// when all tests are run, use unique artifacts directories per test to preserve artifacts from other tests
				if testVisual {
					thisTest = append(thisTest, "--output=tests/e2e/test-artifacts/"+testname)
				}
				thisTest = append(thisTest, path)
				cmd := exec.Command(runArgs[0], thisTest...)
				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, fmt.Sprintf("GITEA_URL=%s", setting.AppURL))

				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				err := cmd.Run()
				if err != nil && !testVisual {
					log.Fatal("Playwright Failed: %s", err)
				}
			})
		})
	}
}
