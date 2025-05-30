// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package notify

import (
	"net/http"
	"strings"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	api "forgejo.org/modules/structs"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
)

// NewAvailable check if unread notifications exist
func NewAvailable(ctx *context.APIContext) {
	// swagger:operation GET /notifications/new notification notifyNewAvailable
	// ---
	// summary: Check if unread notifications exist
	// responses:
	//   "200":
	//     "$ref": "#/responses/NotificationCount"

	total, err := db.Count[activities_model.Notification](ctx, activities_model.FindNotificationOptions{
		UserID: ctx.Doer.ID,
		Status: []activities_model.NotificationStatus{activities_model.NotificationStatusUnread},
	})
	if err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "db.Count[activities_model.Notification]", err)
		return
	}

	ctx.JSON(http.StatusOK, api.NotificationCount{New: total})
}

func getFindNotificationOptions(ctx *context.APIContext) *activities_model.FindNotificationOptions {
	before, since, err := context.GetQueryBeforeSince(ctx.Base)
	if err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "GetQueryBeforeSince", err)
		return nil
	}
	opts := &activities_model.FindNotificationOptions{
		ListOptions:       utils.GetListOptions(ctx),
		UserID:            ctx.Doer.ID,
		UpdatedBeforeUnix: before,
		UpdatedAfterUnix:  since,
	}
	if !ctx.FormBool("all") {
		statuses := ctx.FormStrings("status-types")
		opts.Status = statusStringsToNotificationStatuses(statuses, []string{"unread", "pinned"})
	}

	subjectTypes := ctx.FormStrings("subject-type")
	if len(subjectTypes) != 0 {
		opts.Source = subjectToSource(subjectTypes)
	}

	return opts
}

func subjectToSource(value []string) (result []activities_model.NotificationSource) {
	for _, v := range value {
		switch strings.ToLower(v) {
		case "issue":
			result = append(result, activities_model.NotificationSourceIssue)
		case "pull":
			result = append(result, activities_model.NotificationSourcePullRequest)
		case "commit":
			result = append(result, activities_model.NotificationSourceCommit)
		case "repository":
			result = append(result, activities_model.NotificationSourceRepository)
		}
	}
	return result
}
