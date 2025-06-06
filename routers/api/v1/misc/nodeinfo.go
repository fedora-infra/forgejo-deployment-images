// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"
	"time"

	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/services/context"
)

const cacheKeyNodeInfoUsage = "API_NodeInfoUsage"

// NodeInfo returns the NodeInfo for the Forgejo instance to allow for federation
func NodeInfo(ctx *context.APIContext) {
	// swagger:operation GET /nodeinfo miscellaneous getNodeInfo
	// ---
	// summary: Returns the nodeinfo of the Forgejo application
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/NodeInfo"

	nodeInfoUsage := structs.NodeInfoUsage{}
	if setting.Federation.ShareUserStatistics {
		var cached bool
		nodeInfoUsage, cached = ctx.Cache.Get(cacheKeyNodeInfoUsage).(structs.NodeInfoUsage)

		if !cached {
			usersTotal := int(user_model.CountUsers(ctx, nil))
			now := time.Now()
			timeOneMonthAgo := now.AddDate(0, -1, 0).Unix()
			timeHaveYearAgo := now.AddDate(0, -6, 0).Unix()
			usersActiveMonth := int(user_model.CountUsers(ctx, &user_model.CountUserFilter{LastLoginSince: &timeOneMonthAgo}))
			usersActiveHalfyear := int(user_model.CountUsers(ctx, &user_model.CountUserFilter{LastLoginSince: &timeHaveYearAgo}))

			allIssues, _ := issues_model.CountIssues(ctx, &issues_model.IssuesOptions{})
			allComments, _ := issues_model.CountComments(ctx, &issues_model.FindCommentsOptions{})

			nodeInfoUsage = structs.NodeInfoUsage{
				Users: structs.NodeInfoUsageUsers{
					Total:          usersTotal,
					ActiveMonth:    usersActiveMonth,
					ActiveHalfyear: usersActiveHalfyear,
				},
				LocalPosts:    int(allIssues),
				LocalComments: int(allComments),
			}

			if err := ctx.Cache.Put(cacheKeyNodeInfoUsage, nodeInfoUsage, 180); err != nil {
				ctx.InternalServerError(err)
				return
			}
		}
	}

	nodeInfo := &structs.NodeInfo{
		Version: "2.1",
		Software: structs.NodeInfoSoftware{
			Name:       "forgejo",
			Version:    setting.AppVer,
			Repository: "https://codeberg.org/forgejo/forgejo.git",
			Homepage:   "https://forgejo.org/",
		},
		Protocols: []string{"activitypub"},
		Services: structs.NodeInfoServices{
			Inbound:  []string{},
			Outbound: []string{"rss2.0"},
		},
		OpenRegistrations: setting.Service.ShowRegistrationButton,
		Usage:             nodeInfoUsage,
	}
	ctx.JSON(http.StatusOK, nodeInfo)
}
