// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"fmt"
	"net/http"

	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	shared "forgejo.org/routers/web/shared/packages"
	shared_user "forgejo.org/routers/web/shared/user"
	"forgejo.org/services/context"
)

const (
	tplSettingsPackages            base.TplName = "org/settings/packages"
	tplSettingsPackagesRuleEdit    base.TplName = "org/settings/packages_cleanup_rules_edit"
	tplSettingsPackagesRulePreview base.TplName = "org/settings/packages_cleanup_rules_preview"
)

func Packages(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	err := shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	shared.SetPackagesContext(ctx, ctx.ContextUser)

	ctx.HTML(http.StatusOK, tplSettingsPackages)
}

func PackagesRuleAdd(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	err := shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	shared.SetRuleAddContext(ctx)

	ctx.HTML(http.StatusOK, tplSettingsPackagesRuleEdit)
}

func PackagesRuleEdit(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	err := shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	shared.SetRuleEditContext(ctx, ctx.ContextUser)

	ctx.HTML(http.StatusOK, tplSettingsPackagesRuleEdit)
}

func PackagesRuleAddPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	shared.PerformRuleAddPost(
		ctx,
		ctx.ContextUser,
		fmt.Sprintf("%s/org/%s/settings/packages", setting.AppSubURL, ctx.ContextUser.Name),
		tplSettingsPackagesRuleEdit,
	)
}

func PackagesRuleEditPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	shared.PerformRuleEditPost(
		ctx,
		ctx.ContextUser,
		fmt.Sprintf("%s/org/%s/settings/packages", setting.AppSubURL, ctx.ContextUser.Name),
		tplSettingsPackagesRuleEdit,
	)
}

func PackagesRulePreview(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	err := shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	shared.SetRulePreviewContext(ctx, ctx.ContextUser)

	ctx.HTML(http.StatusOK, tplSettingsPackagesRulePreview)
}

func InitializeCargoIndex(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	shared.InitializeCargoIndex(ctx, ctx.ContextUser)

	ctx.Redirect(fmt.Sprintf("%s/org/%s/settings/packages", setting.AppSubURL, ctx.ContextUser.Name))
}

func RebuildCargoIndex(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsPackages"] = true

	shared.RebuildCargoIndex(ctx, ctx.ContextUser)

	ctx.Redirect(fmt.Sprintf("%s/org/%s/settings/packages", setting.AppSubURL, ctx.ContextUser.Name))
}
