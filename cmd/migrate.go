// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/models/migrations"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"github.com/urfave/cli/v2"
)

// CmdMigrate represents the available migrate sub-command.
var CmdMigrate = &cli.Command{
	Name:        "migrate",
	Usage:       "Migrate the database",
	Description: "This is a command for migrating the database, so that you can run 'forgejo admin user create' before starting the server.",
	Action:      runMigrate,
}

func runMigrate(ctx *cli.Context) error {
	stdCtx, cancel := installSignals()
	defer cancel()

	if err := initDB(stdCtx); err != nil {
		return err
	}

	log.Info("AppPath: %s", setting.AppPath)
	log.Info("AppWorkPath: %s", setting.AppWorkPath)
	log.Info("Custom path: %s", setting.CustomPath)
	log.Info("Log path: %s", setting.Log.RootPath)
	log.Info("Configuration file: %s", setting.CustomConf)

	if err := db.InitEngineWithMigration(context.Background(), migrations.Migrate); err != nil {
		log.Fatal("Failed to initialize ORM engine: %v", err)
		return err
	}

	return nil
}
