// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"regexp"
	"strings"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
	secret_service "code.gitea.io/gitea/services/secrets"
)

func CreateVariable(ctx context.Context, ownerID, repoID int64, name, data string) (*actions_model.ActionVariable, error) {
	if err := secret_service.ValidateName(name); err != nil {
		return nil, err
	}

	if err := envNameCIRegexMatch(name); err != nil {
		return nil, err
	}

	v, err := actions_model.InsertVariable(ctx, ownerID, repoID, name, util.ReserveLineBreakForTextarea(data))
	if err != nil {
		return nil, err
	}

	return v, nil
}

func UpdateVariable(ctx context.Context, variableID, ownerID, repoID int64, name, data string) (bool, error) {
	if err := secret_service.ValidateName(name); err != nil {
		return false, err
	}

	if err := envNameCIRegexMatch(name); err != nil {
		return false, err
	}

	return actions_model.UpdateVariable(ctx, &actions_model.ActionVariable{
		ID:      variableID,
		Name:    strings.ToUpper(name),
		Data:    util.ReserveLineBreakForTextarea(data),
		OwnerID: ownerID,
		RepoID:  repoID,
	})
}

// some regular expression of `variables` and `secrets`
// reference to:
// https://docs.github.com/en/actions/learn-github-actions/variables#naming-conventions-for-configuration-variables
// https://docs.github.com/en/actions/security-guides/encrypted-secrets#naming-your-secrets
var (
	forbiddenEnvNameCIRx = regexp.MustCompile("(?i)^CI")
)

func envNameCIRegexMatch(name string) error {
	if forbiddenEnvNameCIRx.MatchString(name) {
		log.Error("Env Name cannot be ci")
		return util.NewInvalidArgumentErrorf("env name cannot be ci")
	}
	return nil
}
