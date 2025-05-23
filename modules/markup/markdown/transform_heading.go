// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"fmt"

	"forgejo.org/modules/markup"
	mdutil "forgejo.org/modules/markup/markdown/util"
	"forgejo.org/modules/util"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func (g *ASTTransformer) transformHeading(_ *markup.RenderContext, v *ast.Heading, reader text.Reader, tocList *[]markup.Header) {
	for _, attr := range v.Attributes() {
		if _, ok := attr.Value.([]byte); !ok {
			v.SetAttribute(attr.Name, []byte(fmt.Sprintf("%v", attr.Value)))
		}
	}
	txt := mdutil.Text(v, reader.Source())
	header := markup.Header{
		Text:  util.UnsafeBytesToString(txt),
		Level: v.Level,
	}
	if id, found := v.AttributeString("id"); found {
		header.ID = util.UnsafeBytesToString(id.([]byte))
	}
	*tocList = append(*tocList, header)
	g.applyElementDir(v)
}
