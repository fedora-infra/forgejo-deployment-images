// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"fmt"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	secret_model "forgejo.org/models/secret"

	runnerv1 "code.gitea.io/actions-proto-go/runner/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func PickTask(ctx context.Context, runner *actions_model.ActionRunner) (*runnerv1.Task, bool, error) {
	var (
		task *runnerv1.Task
		job  *actions_model.ActionRunJob
	)

	if err := db.WithTx(ctx, func(ctx context.Context) error {
		t, ok, err := actions_model.CreateTaskForRunner(ctx, runner)
		if err != nil {
			return fmt.Errorf("CreateTaskForRunner: %w", err)
		}
		if !ok {
			return nil
		}

		if err := t.LoadAttributes(ctx); err != nil {
			return fmt.Errorf("task LoadAttributes: %w", err)
		}
		job = t.Job

		secrets, err := secret_model.GetSecretsOfTask(ctx, t)
		if err != nil {
			return fmt.Errorf("GetSecretsOfTask: %w", err)
		}

		vars, err := actions_model.GetVariablesOfRun(ctx, t.Job.Run)
		if err != nil {
			return fmt.Errorf("GetVariablesOfRun: %w", err)
		}

		needs, err := findTaskNeeds(ctx, job)
		if err != nil {
			return fmt.Errorf("findTaskNeeds: %w", err)
		}

		taskContext, err := generateTaskContext(t)
		if err != nil {
			return fmt.Errorf("generateTaskContext: %w", err)
		}

		task = &runnerv1.Task{
			Id:              t.ID,
			WorkflowPayload: t.Job.WorkflowPayload,
			Context:         taskContext,
			Secrets:         secrets,
			Vars:            vars,
			Needs:           needs,
		}

		return nil
	}); err != nil {
		return nil, false, err
	}

	if task == nil {
		return nil, false, nil
	}

	CreateCommitStatus(ctx, job)

	return task, true, nil
}

func generateTaskContext(t *actions_model.ActionTask) (*structpb.Struct, error) {
	giteaRuntimeToken, err := CreateAuthorizationToken(t.ID, t.Job.RunID, t.JobID)
	if err != nil {
		return nil, err
	}

	gitCtx := GenerateGiteaContext(t.Job.Run, t.Job)
	gitCtx["token"] = t.Token
	gitCtx["gitea_runtime_token"] = giteaRuntimeToken

	return structpb.NewStruct(gitCtx)
}

func findTaskNeeds(ctx context.Context, taskJob *actions_model.ActionRunJob) (map[string]*runnerv1.TaskNeed, error) {
	taskNeeds, err := FindTaskNeeds(ctx, taskJob)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]*runnerv1.TaskNeed, len(taskNeeds))
	for jobID, taskNeed := range taskNeeds {
		ret[jobID] = &runnerv1.TaskNeed{
			Outputs: taskNeed.Outputs,
			Result:  runnerv1.Result(taskNeed.Result),
		}
	}
	return ret, nil
}
