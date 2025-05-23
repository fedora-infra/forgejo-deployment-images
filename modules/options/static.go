// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build bindata

package options

import (
	"forgejo.org/modules/assetfs"
)

func BuiltinAssets() *assetfs.Layer {
	return assetfs.Bindata("builtin(bindata)", Assets)
}
