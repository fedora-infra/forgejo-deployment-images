// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4

type webfingerJRD struct {
	Subject    string           `json:"subject,omitempty"`
	Aliases    []string         `json:"aliases,omitempty"`
	Properties map[string]any   `json:"properties,omitempty"`
	Links      []*webfingerLink `json:"links,omitempty"`
}

type webfingerLink struct {
	Rel        string            `json:"rel,omitempty"`
	Type       string            `json:"type,omitempty"`
	Href       string            `json:"href,omitempty"`
	Titles     map[string]string `json:"titles,omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
}

// WebfingerQuery returns information about a resource
// https://datatracker.ietf.org/doc/html/rfc7565
func WebfingerQuery(ctx *context.Context) {
	appURL, _ := url.Parse(setting.AppURL)

	resource, err := url.Parse(ctx.FormTrim("resource"))
	if err != nil {
		ctx.Error(http.StatusBadRequest)
		return
	}

	var u *user_model.User

	switch resource.Scheme {
	case "acct":
		// allow only the current host
		parts := strings.SplitN(resource.Opaque, "@", 2)
		if len(parts) != 2 {
			ctx.Error(http.StatusBadRequest)
			return
		}
		if parts[1] != appURL.Host {
			ctx.Error(http.StatusBadRequest)
			return
		}

		u, err = user_model.GetUserByName(ctx, parts[0])
	case "mailto":
		u, err = user_model.GetUserByEmail(ctx, resource.Opaque)
		if u != nil && u.KeepEmailPrivate {
			err = user_model.ErrUserNotExist{}
		}
	case "https", "http":
		if resource.Host != appURL.Host {
			ctx.Error(http.StatusBadRequest)
			return
		}

		p := strings.Trim(resource.Path, "/")
		if len(p) == 0 {
			ctx.Error(http.StatusNotFound)
			return
		}

		parts := strings.Split(p, "/")

		switch len(parts) {
		case 1: // user
			u, err = user_model.GetUserByName(ctx, parts[0])
		case 2: // repository
			ctx.Error(http.StatusNotFound)
			return

		case 3:
			switch parts[2] {
			case "issues":
				ctx.Error(http.StatusNotFound)
				return

			case "pulls":
				ctx.Error(http.StatusNotFound)
				return

			case "projects":
				ctx.Error(http.StatusNotFound)
				return

			default:
				ctx.Error(http.StatusNotFound)
				return
			}

		default:
			ctx.Error(http.StatusNotFound)
			return
		}

	default:
		ctx.Error(http.StatusBadRequest)
		return
	}
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusNotFound)
		} else {
			log.Error("Error getting user: %s Error: %v", resource.Opaque, err)
			ctx.Error(http.StatusInternalServerError)
		}
		return
	}

	if !user_model.IsUserVisibleToViewer(ctx, u, ctx.Doer) {
		ctx.Error(http.StatusNotFound)
		return
	}

	aliases := []string{
		u.HTMLURL(),
		appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(u.ID),
	}
	if !u.KeepEmailPrivate {
		aliases = append(aliases, fmt.Sprintf("mailto:%s", u.Email))
	}

	links := []*webfingerLink{
		{
			Rel:  "http://webfinger.net/rel/profile-page",
			Type: "text/html",
			Href: u.HTMLURL(),
		},
		{
			Rel:  "http://webfinger.net/rel/avatar",
			Href: u.AvatarLink(ctx),
		},
		{
			Rel:  "self",
			Type: "application/activity+json",
			Href: appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(u.ID),
		},
		{
			Rel:  "http://openid.net/specs/connect/1.0/issuer",
			Href: appURL.String(),
		},
	}

	ctx.Resp.Header().Add("Access-Control-Allow-Origin", "*")
	ctx.JSON(http.StatusOK, &webfingerJRD{
		Subject: fmt.Sprintf("acct:%s@%s", url.QueryEscape(u.Name), appURL.Host),
		Aliases: aliases,
		Links:   links,
	})
	ctx.Resp.Header().Set("Content-Type", "application/jrd+json")
}
