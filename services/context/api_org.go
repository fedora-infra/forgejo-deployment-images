// Copyright 2016 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import "forgejo.org/models/organization"

// APIOrganization contains organization and team
type APIOrganization struct {
	Organization *organization.Organization
	Team         *organization.Team
}
