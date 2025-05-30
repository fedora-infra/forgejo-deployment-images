// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"testing"

	"forgejo.org/modules/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlags(t *testing.T) {
	assert.EqualValues(t, Ldefault, Flags{}.Bits())
	assert.EqualValues(t, 0, FlagsFromString("").Bits())
	assert.EqualValues(t, Lgopid, FlagsFromString("", Lgopid).Bits())
	assert.EqualValues(t, 0, FlagsFromString("none", Lgopid).Bits())
	assert.EqualValues(t, Ldate|Ltime, FlagsFromString("date,time", Lgopid).Bits())

	assert.EqualValues(t, "stdflags", FlagsFromString("stdflags").String())
	assert.EqualValues(t, "medfile", FlagsFromString("medfile").String())

	bs, err := json.Marshal(FlagsFromString("utc,level"))
	require.NoError(t, err)
	assert.EqualValues(t, `"level,utc"`, string(bs))
	var flags Flags
	require.NoError(t, json.Unmarshal(bs, &flags))
	assert.EqualValues(t, LUTC|Llevel, flags.Bits())
}
