// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package console

import (
	"strings"
	"testing"

	"forgejo.org/modules/git"
	"forgejo.org/modules/markup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderConsole(t *testing.T) {
	var render Renderer
	kases := map[string]string{
		"\x1b[37m\x1b[40mnpm\x1b[0m \x1b[0m\x1b[32minfo\x1b[0m \x1b[0m\x1b[35mit worked if it ends with\x1b[0m ok": "<span class=\"term-fg37 term-bg40\">npm</span> <span class=\"term-fg32\">info</span> <span class=\"term-fg35\">it worked if it ends with</span> ok",
	}

	for k, v := range kases {
		var buf strings.Builder
		canRender := render.CanRender("test", strings.NewReader(k))
		assert.True(t, canRender)

		err := render.Render(&markup.RenderContext{Ctx: git.DefaultContext},
			strings.NewReader(k), &buf)
		require.NoError(t, err)
		assert.EqualValues(t, v, buf.String())
	}
}
