// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	"forgejo.org/models/webhook"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
	webhook_service "forgejo.org/services/webhook"
)

const (
	// tplAdminHooks template path to render hook settings
	tplAdminHooks base.TplName = "admin/hooks"
)

// DefaultOrSystemWebhooks renders both admin default and system webhook list pages
func DefaultOrSystemWebhooks(ctx *context.Context) {
	var err error

	ctx.Data["Title"] = ctx.Tr("admin.hooks")
	ctx.Data["PageIsAdminSystemHooks"] = true
	ctx.Data["PageIsAdminDefaultHooks"] = true

	def := make(map[string]any, len(ctx.Data))
	sys := make(map[string]any, len(ctx.Data))
	for k, v := range ctx.Data {
		def[k] = v
		sys[k] = v
	}

	sys["Title"] = ctx.Tr("admin.systemhooks")
	sys["Description"] = ctx.Tr("admin.systemhooks.desc", "https://forgejo.org/docs/latest/user/webhooks/")
	sys["Webhooks"], err = webhook.GetSystemWebhooks(ctx, false)
	sys["BaseLink"] = setting.AppSubURL + "/admin/hooks"
	sys["BaseLinkNew"] = setting.AppSubURL + "/admin/system-hooks"
	sys["WebhookList"] = webhook_service.List()
	if err != nil {
		ctx.ServerError("GetWebhooksAdmin", err)
		return
	}

	def["Title"] = ctx.Tr("admin.defaulthooks")
	def["Description"] = ctx.Tr("admin.defaulthooks.desc", "https://forgejo.org/docs/latest/user/webhooks/")
	def["Webhooks"], err = webhook.GetDefaultWebhooks(ctx)
	def["BaseLink"] = setting.AppSubURL + "/admin/hooks"
	def["BaseLinkNew"] = setting.AppSubURL + "/admin/default-hooks"
	def["WebhookList"] = webhook_service.List()
	if err != nil {
		ctx.ServerError("GetWebhooksAdmin", err)
		return
	}

	ctx.Data["DefaultWebhooks"] = def
	ctx.Data["SystemWebhooks"] = sys

	ctx.HTML(http.StatusOK, tplAdminHooks)
}

// DeleteDefaultOrSystemWebhook handler to delete an admin-defined system or default webhook
func DeleteDefaultOrSystemWebhook(ctx *context.Context) {
	if err := webhook.DeleteDefaultSystemWebhook(ctx, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteDefaultWebhook: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("repo.settings.webhook_deletion_success"))
	}

	ctx.JSONRedirect(setting.AppSubURL + "/admin/hooks")
}
