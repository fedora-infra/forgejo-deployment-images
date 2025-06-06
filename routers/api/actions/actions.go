// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"net/http"

	"forgejo.org/modules/web"
	"forgejo.org/routers/api/actions/ping"
	"forgejo.org/routers/api/actions/runner"
)

func Routes(prefix string) *web.Route {
	m := web.NewRoute()

	path, handler := ping.NewPingServiceHandler()
	m.Post(path+"*", http.StripPrefix(prefix, handler).ServeHTTP)

	path, handler = runner.NewRunnerServiceHandler()
	m.Post(path+"*", http.StripPrefix(prefix, handler).ServeHTTP)

	return m
}
