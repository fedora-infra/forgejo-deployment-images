// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"
	"net/http"
	"slices"

	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"
	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/base"
	"forgejo.org/modules/container"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	alpine_module "forgejo.org/modules/packages/alpine"
	arch_model "forgejo.org/modules/packages/arch"
	debian_module "forgejo.org/modules/packages/debian"
	rpm_module "forgejo.org/modules/packages/rpm"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	packages_helper "forgejo.org/routers/api/packages/helper"
	shared_user "forgejo.org/routers/web/shared/user"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
	packages_service "forgejo.org/services/packages"
)

const (
	tplPackagesList       base.TplName = "user/overview/packages"
	tplPackagesView       base.TplName = "package/view"
	tplPackageVersionList base.TplName = "user/overview/package_versions"
	tplPackagesSettings   base.TplName = "package/settings"
)

// ListPackages displays a list of all packages of the context user
func ListPackages(ctx *context.Context) {
	shared_user.PrepareContextForProfileBigAvatar(ctx)
	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	query := ctx.FormTrim("q")
	packageType := ctx.FormTrim("type")

	pvs, total, err := packages_model.SearchLatestVersions(ctx, &packages_model.PackageSearchOptions{
		Paginator: &db.ListOptions{
			PageSize: setting.UI.PackagesPagingNum,
			Page:     page,
		},
		OwnerID:    ctx.ContextUser.ID,
		Type:       packages_model.Type(packageType),
		Name:       packages_model.SearchValue{Value: query},
		IsInternal: optional.Some(false),
	})
	if err != nil {
		ctx.ServerError("SearchLatestVersions", err)
		return
	}

	pds, err := packages_model.GetPackageDescriptors(ctx, pvs)
	if err != nil {
		ctx.ServerError("GetPackageDescriptors", err)
		return
	}

	repositoryAccessMap := make(map[int64]bool)
	for _, pd := range pds {
		if pd.Repository == nil {
			continue
		}
		if _, has := repositoryAccessMap[pd.Repository.ID]; has {
			continue
		}

		permission, err := access_model.GetUserRepoPermission(ctx, pd.Repository, ctx.Doer)
		if err != nil {
			ctx.ServerError("GetUserRepoPermission", err)
			return
		}
		repositoryAccessMap[pd.Repository.ID] = permission.HasAccess()
	}

	hasPackages, err := packages_model.HasOwnerPackages(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.ServerError("HasOwnerPackages", err)
		return
	}

	shared_user.RenderUserHeader(ctx)

	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["IsPackagesPage"] = true
	ctx.Data["Query"] = query
	ctx.Data["PackageType"] = packageType
	ctx.Data["AvailableTypes"] = packages_model.TypeList
	ctx.Data["HasPackages"] = hasPackages
	ctx.Data["PackageDescriptors"] = pds
	ctx.Data["Total"] = total
	ctx.Data["RepositoryAccessMap"] = repositoryAccessMap

	err = shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	// TODO: context/org -> HandleOrgAssignment() can not be used
	if ctx.ContextUser.IsOrganization() {
		org := org_model.OrgFromUser(ctx.ContextUser)
		ctx.Data["Org"] = org
		ctx.Data["OrgLink"] = ctx.ContextUser.OrganisationLink()

		if ctx.Doer != nil {
			ctx.Data["IsOrganizationMember"], _ = org_model.IsOrganizationMember(ctx, org.ID, ctx.Doer.ID)
			ctx.Data["IsOrganizationOwner"], _ = org_model.IsOrganizationOwner(ctx, org.ID, ctx.Doer.ID)
		} else {
			ctx.Data["IsOrganizationMember"] = false
			ctx.Data["IsOrganizationOwner"] = false
		}
	}

	pager := context.NewPagination(int(total), setting.UI.PackagesPagingNum, page, 5)
	pager.AddParam(ctx, "q", "Query")
	pager.AddParam(ctx, "type", "PackageType")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplPackagesList)
}

// RedirectToLastVersion redirects to the latest package version
func RedirectToLastVersion(ctx *context.Context) {
	p, err := packages_model.GetPackageByName(ctx, ctx.Package.Owner.ID, packages_model.Type(ctx.Params("type")), ctx.Params("name"))
	if err != nil {
		if err == packages_model.ErrPackageNotExist {
			ctx.NotFound("GetPackageByName", err)
		} else {
			ctx.ServerError("GetPackageByName", err)
		}
		return
	}

	pvs, _, err := packages_model.SearchLatestVersions(ctx, &packages_model.PackageSearchOptions{
		PackageID:  p.ID,
		IsInternal: optional.Some(false),
	})
	if err != nil {
		ctx.ServerError("GetPackageByName", err)
		return
	}
	if len(pvs) == 0 {
		ctx.NotFound("", err)
		return
	}

	pd, err := packages_model.GetPackageDescriptor(ctx, pvs[0])
	if err != nil {
		ctx.ServerError("GetPackageDescriptor", err)
		return
	}

	ctx.Redirect(pd.VersionWebLink())
}

// ViewPackageVersion displays a single package version
func ViewPackageVersion(ctx *context.Context) {
	pd := ctx.Package.Descriptor

	shared_user.RenderUserHeader(ctx)

	ctx.Data["Title"] = pd.Package.Name
	ctx.Data["IsPackagesPage"] = true
	ctx.Data["PackageDescriptor"] = pd
	ctx.Data["PackageRegistryHost"] = setting.Packages.RegistryHost

	switch pd.Package.Type {
	case packages_model.TypeContainer:

	case packages_model.TypeAlpine:
		branches := make(container.Set[string])
		repositories := make(container.Set[string])
		architectures := make(container.Set[string])

		for _, f := range pd.Files {
			for _, pp := range f.Properties {
				switch pp.Name {
				case alpine_module.PropertyBranch:
					branches.Add(pp.Value)
				case alpine_module.PropertyRepository:
					repositories.Add(pp.Value)
				case alpine_module.PropertyArchitecture:
					architectures.Add(pp.Value)
				}
			}
		}

		ctx.Data["Branches"] = slices.Sorted(branches.Seq())
		ctx.Data["Repositories"] = slices.Sorted(repositories.Seq())
		ctx.Data["Architectures"] = slices.Sorted(architectures.Seq())
	case packages_model.TypeArch:
		ctx.Data["SignMail"] = fmt.Sprintf("%s@noreply.%s", ctx.Package.Owner.Name, setting.Packages.RegistryHost)
		groups := make(container.Set[string])
		for _, f := range pd.Files {
			for _, pp := range f.Properties {
				if pp.Name == arch_model.PropertyDistribution {
					groups.Add(pp.Value)
				}
			}
		}
		ctx.Data["Groups"] = slices.Sorted(groups.Seq())
	case packages_model.TypeDebian:
		distributions := make(container.Set[string])
		components := make(container.Set[string])
		architectures := make(container.Set[string])

		for _, f := range pd.Files {
			for _, pp := range f.Properties {
				switch pp.Name {
				case debian_module.PropertyDistribution:
					distributions.Add(pp.Value)
				case debian_module.PropertyComponent:
					components.Add(pp.Value)
				case debian_module.PropertyArchitecture:
					architectures.Add(pp.Value)
				}
			}
		}

		ctx.Data["Distributions"] = slices.Sorted(distributions.Seq())
		ctx.Data["Components"] = slices.Sorted(components.Seq())
		ctx.Data["Architectures"] = slices.Sorted(architectures.Seq())
	case packages_model.TypeRpm, packages_model.TypeAlt:
		groups := make(container.Set[string])
		architectures := make(container.Set[string])

		for _, f := range pd.Files {
			for _, pp := range f.Properties {
				switch pp.Name {
				case rpm_module.PropertyGroup:
					groups.Add(pp.Value)
				case rpm_module.PropertyArchitecture:
					architectures.Add(pp.Value)
				}
			}
		}

		ctx.Data["Groups"] = slices.Sorted(groups.Seq())
		ctx.Data["Architectures"] = slices.Sorted(architectures.Seq())
	}

	var (
		total int64
		pvs   []*packages_model.PackageVersion
		err   error
	)
	switch pd.Package.Type {
	case packages_model.TypeContainer:
		pvs, total, err = container_model.SearchImageTags(ctx, &container_model.ImageTagsSearchOptions{
			Paginator: db.NewAbsoluteListOptions(0, 5),
			PackageID: pd.Package.ID,
			IsTagged:  true,
		})
	default:
		pvs, total, err = packages_model.SearchVersions(ctx, &packages_model.PackageSearchOptions{
			Paginator:  db.NewAbsoluteListOptions(0, 5),
			PackageID:  pd.Package.ID,
			IsInternal: optional.Some(false),
		})
	}
	if err != nil {
		ctx.ServerError("", err)
		return
	}

	ctx.Data["LatestVersions"] = pvs
	ctx.Data["TotalVersionCount"] = total

	ctx.Data["CanWritePackages"] = ctx.Package.AccessMode >= perm.AccessModeWrite || ctx.IsUserSiteAdmin()

	hasRepositoryAccess := false
	if pd.Repository != nil {
		permission, err := access_model.GetUserRepoPermission(ctx, pd.Repository, ctx.Doer)
		if err != nil {
			ctx.ServerError("GetUserRepoPermission", err)
			return
		}
		hasRepositoryAccess = permission.HasAccess()
	}
	ctx.Data["HasRepositoryAccess"] = hasRepositoryAccess

	err = shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	ctx.HTML(http.StatusOK, tplPackagesView)
}

// ListPackageVersions lists all versions of a package
func ListPackageVersions(ctx *context.Context) {
	shared_user.PrepareContextForProfileBigAvatar(ctx)
	p, err := packages_model.GetPackageByName(ctx, ctx.Package.Owner.ID, packages_model.Type(ctx.Params("type")), ctx.Params("name"))
	if err != nil {
		if err == packages_model.ErrPackageNotExist {
			ctx.NotFound("GetPackageByName", err)
		} else {
			ctx.ServerError("GetPackageByName", err)
		}
		return
	}

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	pagination := &db.ListOptions{
		PageSize: setting.UI.PackagesPagingNum,
		Page:     page,
	}

	query := ctx.FormTrim("q")
	sort := ctx.FormTrim("sort")

	shared_user.RenderUserHeader(ctx)

	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["IsPackagesPage"] = true
	ctx.Data["PackageDescriptor"] = &packages_model.PackageDescriptor{
		Package: p,
		Owner:   ctx.Package.Owner,
	}
	ctx.Data["Query"] = query
	ctx.Data["Sort"] = sort

	pagerParams := map[string]string{
		"q":    query,
		"sort": sort,
	}

	var (
		total int64
		pvs   []*packages_model.PackageVersion
	)
	switch p.Type {
	case packages_model.TypeContainer:
		tagged := ctx.FormTrim("tagged")

		pagerParams["tagged"] = tagged
		ctx.Data["Tagged"] = tagged

		pvs, total, err = container_model.SearchImageTags(ctx, &container_model.ImageTagsSearchOptions{
			Paginator: pagination,
			PackageID: p.ID,
			Query:     query,
			IsTagged:  tagged == "" || tagged == "tagged",
			Sort:      sort,
		})
		if err != nil {
			ctx.ServerError("SearchImageTags", err)
			return
		}
	default:
		pvs, total, err = packages_model.SearchVersions(ctx, &packages_model.PackageSearchOptions{
			Paginator: pagination,
			PackageID: p.ID,
			Version: packages_model.SearchValue{
				ExactMatch: false,
				Value:      query,
			},
			IsInternal: optional.Some(false),
			Sort:       sort,
		})
		if err != nil {
			ctx.ServerError("SearchVersions", err)
			return
		}
	}

	ctx.Data["PackageDescriptors"], err = packages_model.GetPackageDescriptors(ctx, pvs)
	if err != nil {
		ctx.ServerError("GetPackageDescriptors", err)
		return
	}

	ctx.Data["Total"] = total

	err = shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	pager := context.NewPagination(int(total), setting.UI.PackagesPagingNum, page, 5)
	for k, v := range pagerParams {
		pager.AddParamString(k, v)
	}
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplPackageVersionList)
}

// PackageSettings displays the package settings page
func PackageSettings(ctx *context.Context) {
	pd := ctx.Package.Descriptor

	shared_user.RenderUserHeader(ctx)

	ctx.Data["Title"] = pd.Package.Name
	ctx.Data["IsPackagesPage"] = true
	ctx.Data["PackageDescriptor"] = pd

	repos, _, _ := repo_model.GetUserRepositories(ctx, &repo_model.SearchRepoOptions{
		Actor:   pd.Owner,
		Private: true,
	})
	ctx.Data["Repos"] = repos
	ctx.Data["CanWritePackages"] = ctx.Package.AccessMode >= perm.AccessModeWrite || ctx.IsUserSiteAdmin()

	err := shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	ctx.HTML(http.StatusOK, tplPackagesSettings)
}

// PackageSettingsPost updates the package settings
func PackageSettingsPost(ctx *context.Context) {
	pd := ctx.Package.Descriptor

	form := web.GetForm(ctx).(*forms.PackageSettingForm)
	switch form.Action {
	case "link":
		success := func() bool {
			repoID := int64(0)
			if form.RepoID != 0 {
				repo, err := repo_model.GetRepositoryByID(ctx, form.RepoID)
				if err != nil {
					log.Error("Error getting repository: %v", err)
					return false
				}

				if repo.OwnerID != pd.Owner.ID {
					return false
				}

				repoID = repo.ID
			}

			if err := packages_model.SetRepositoryLink(ctx, pd.Package.ID, repoID); err != nil {
				log.Error("Error updating package: %v", err)
				return false
			}

			return true
		}()

		if success {
			ctx.Flash.Success(ctx.Tr("packages.settings.link.success"))
		} else {
			ctx.Flash.Error(ctx.Tr("packages.settings.link.error"))
		}

		ctx.Redirect(ctx.Link)
		return
	case "delete":
		err := packages_service.RemovePackageVersion(ctx, ctx.Doer, ctx.Package.Descriptor.Version)
		if err != nil {
			log.Error("Error deleting package: %v", err)
			ctx.Flash.Error(ctx.Tr("packages.settings.delete.error"))
		} else {
			ctx.Flash.Success(ctx.Tr("packages.settings.delete.success"))
		}

		redirectURL := ctx.Package.Owner.HomeLink() + "/-/packages"
		// redirect to the package if there are still versions available
		if has, _ := packages_model.ExistVersion(ctx, &packages_model.PackageSearchOptions{PackageID: ctx.Package.Descriptor.Package.ID, IsInternal: optional.Some(false)}); has {
			redirectURL = ctx.Package.Descriptor.PackageWebLink()
		}

		ctx.Redirect(redirectURL)
		return
	}
}

// DownloadPackageFile serves the content of a package file
func DownloadPackageFile(ctx *context.Context) {
	pf, err := packages_model.GetFileForVersionByID(ctx, ctx.Package.Descriptor.Version.ID, ctx.ParamsInt64(":fileid"))
	if err != nil {
		if err == packages_model.ErrPackageFileNotExist {
			ctx.NotFound("", err)
		} else {
			ctx.ServerError("GetFileForVersionByID", err)
		}
		return
	}

	s, u, _, err := packages_service.GetPackageFileStream(ctx, pf)
	if err != nil {
		ctx.ServerError("GetPackageFileStream", err)
		return
	}

	packages_helper.ServePackageFile(ctx, s, u, pf)
}
