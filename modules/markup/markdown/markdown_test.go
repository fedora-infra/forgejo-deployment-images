// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown_test

import (
	"context"
	"html/template"
	"os"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	AppURL  = "http://localhost:3000/"
	FullURL = AppURL + "gogits/gogs/"
)

// these values should match the const above
var localMetas = map[string]string{
	"user":     "gogits",
	"repo":     "gogs",
	"repoPath": "../../../tests/gitea-repositories-meta/user13/repo11.git/",
}

func TestMain(m *testing.M) {
	unittest.InitSettings()
	if err := git.InitSimple(context.Background()); err != nil {
		log.Fatal("git init failed, err: %v", err)
	}
	markup.Init(&markup.ProcessorHelper{
		IsUsernameMentionable: func(ctx context.Context, username string) bool {
			return username == "r-lyeh"
		},
	})
	os.Exit(m.Run())
}

func TestRender_StandardLinks(t *testing.T) {
	setting.AppURL = AppURL

	test := func(input, expected, expectedWiki string) {
		buffer, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: FullURL,
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(buffer)))

		buffer, err = markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: FullURL,
			},
			IsWiki: true,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expectedWiki), strings.TrimSpace(string(buffer)))
	}

	googleRendered := `<p><a href="https://google.com/" rel="nofollow">https://google.com/</a></p>`
	test("<https://google.com/>", googleRendered, googleRendered)

	lnk := util.URLJoin(FullURL, "WikiPage")
	lnkWiki := util.URLJoin(FullURL, "wiki", "WikiPage")
	test("[WikiPage](WikiPage)",
		`<p><a href="`+lnk+`" rel="nofollow">WikiPage</a></p>`,
		`<p><a href="`+lnkWiki+`" rel="nofollow">WikiPage</a></p>`)
}

func TestRender_Images(t *testing.T) {
	setting.AppURL = AppURL

	test := func(input, expected string) {
		buffer, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: FullURL,
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(buffer)))
	}

	url := "../../.images/src/02/train.jpg"
	title := "Train"
	href := "https://gitea.io"
	result := util.URLJoin(FullURL, url)
	// hint: With Markdown v2.5.2, there is a new syntax: [link](URL){:target="_blank"} , but we do not support it now

	test(
		"!["+title+"]("+url+")",
		`<p><a href="`+result+`" target="_blank" rel="nofollow noopener"><img src="`+result+`" alt="`+title+`"/></a></p>`)

	test(
		"[["+title+"|"+url+"]]",
		`<p><a href="`+result+`" rel="nofollow"><img src="`+result+`" title="`+title+`" alt=""/></a></p>`)
	test(
		"[!["+title+"]("+url+")]("+href+")",
		`<p><a href="`+href+`" rel="nofollow"><img src="`+result+`" alt="`+title+`"/></a></p>`)

	test(
		"!["+title+"]("+url+")",
		`<p><a href="`+result+`" target="_blank" rel="nofollow noopener"><img src="`+result+`" alt="`+title+`"/></a></p>`)

	test(
		"[["+title+"|"+url+"]]",
		`<p><a href="`+result+`" rel="nofollow"><img src="`+result+`" title="`+title+`" alt=""/></a></p>`)
	test(
		"[!["+title+"]("+url+")]("+href+")",
		`<p><a href="`+href+`" rel="nofollow"><img src="`+result+`" alt="`+title+`"/></a></p>`)
}

func testAnswers(baseURLContent, baseURLImages string) []string {
	return []string{
		`<p>Wiki! Enjoy :)</p>
<ul>
<li><a href="` + baseURLContent + `/Links" rel="nofollow">Links, Language bindings, Engine bindings</a></li>
<li><a href="` + baseURLContent + `/Tips" rel="nofollow">Tips</a></li>
</ul>
<p>See commit <a href="/gogits/gogs/commit/65f1bf27bc" rel="nofollow"><code>65f1bf27bc</code></a></p>
<p>Ideas and codes</p>
<ul>
<li>Bezier widget (by <a href="/r-lyeh" class="mention" rel="nofollow">@r-lyeh</a>) <a href="http://localhost:3000/ocornut/imgui/issues/786" class="ref-issue" rel="nofollow">ocornut/imgui#786</a></li>
<li>Bezier widget (by <a href="/r-lyeh" class="mention" rel="nofollow">@r-lyeh</a>) <a href="http://localhost:3000/gogits/gogs/issues/786" class="ref-issue" rel="nofollow">#786</a></li>
<li>Node graph editors <a href="https://github.com/ocornut/imgui/issues/306" rel="nofollow">https://github.com/ocornut/imgui/issues/306</a></li>
<li><a href="` + baseURLContent + `/memory_editor_example" rel="nofollow">Memory Editor</a></li>
<li><a href="` + baseURLContent + `/plot_var_example" rel="nofollow">Plot var helper</a></li>
</ul>
`,
		`<h2 id="user-content-what-is-wine-staging">What is Wine Staging?</h2>
<p><strong>Wine Staging</strong> on website <a href="http://wine-staging.com" rel="nofollow">wine-staging.com</a>.</p>
<h2 id="user-content-quick-links">Quick Links</h2>
<p>Here are some links to the most important topics. You can find the full list of pages at the sidebar.</p>
<table>
<thead>
<tr>
<th><a href="` + baseURLImages + `/images/icon-install.png" rel="nofollow"><img src="` + baseURLImages + `/images/icon-install.png" title="icon-install.png" alt=""/></a></th>
<th><a href="` + baseURLContent + `/Installation" rel="nofollow">Installation</a></th>
</tr>
</thead>
<tbody>
<tr>
<td><a href="` + baseURLImages + `/images/icon-usage.png" rel="nofollow"><img src="` + baseURLImages + `/images/icon-usage.png" title="icon-usage.png" alt=""/></a></td>
<td><a href="` + baseURLContent + `/Usage" rel="nofollow">Usage</a></td>
</tr>
</tbody>
</table>
`,
		`<p><a href="http://www.excelsiorjet.com/" rel="nofollow">Excelsior JET</a> allows you to create native executables for Windows, Linux and Mac OS X.</p>
<ol>
<li><a href="https://github.com/libgdx/libgdx/wiki/Gradle-on-the-Commandline#packaging-for-the-desktop" rel="nofollow">Package your libGDX application</a><br/>
<a href="` + baseURLImages + `/images/1.png" rel="nofollow"><img src="` + baseURLImages + `/images/1.png" title="1.png" alt=""/></a></li>
<li>Perform a test run by hitting the Run! button.<br/>
<a href="` + baseURLImages + `/images/2.png" rel="nofollow"><img src="` + baseURLImages + `/images/2.png" title="2.png" alt=""/></a></li>
</ol>
<h2 id="user-content-custom-id">More tests</h2>
<p>(from <a href="https://www.markdownguide.org/extended-syntax/" rel="nofollow">https://www.markdownguide.org/extended-syntax/</a>)</p>
<h3 id="user-content-checkboxes">Checkboxes</h3>
<ul>
<li class="task-list-item"><input type="checkbox" disabled="" data-source-position="434"/>unchecked</li>
<li class="task-list-item"><input type="checkbox" disabled="" data-source-position="450" checked=""/>checked</li>
<li class="task-list-item"><input type="checkbox" disabled="" data-source-position="464"/>still unchecked</li>
</ul>
<h3 id="user-content-definition-list">Definition list</h3>
<dl>
<dt>First Term</dt>
<dd>This is the definition of the first term.</dd>
<dt>Second Term</dt>
<dd>This is one definition of the second term.</dd>
<dd>This is another definition of the second term.</dd>
</dl>
<h3 id="user-content-footnotes">Footnotes</h3>
<p>Here is a simple footnote,<sup id="fnref:user-content-1"><a href="#fn:user-content-1" rel="nofollow">1</a></sup> and here is a longer one.<sup id="fnref:user-content-bignote"><a href="#fn:user-content-bignote" rel="nofollow">2</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-1">
<p>This is the first footnote. <a href="#fnref:user-content-1" rel="nofollow">↩︎</a></p>
</li>
<li id="fn:user-content-bignote">
<p>Here is one with multiple paragraphs and code.</p>
<p>Indent paragraphs to include them in the footnote.</p>
<p><code>{ my code }</code></p>
<p>Add as many paragraphs as you like. <a href="#fnref:user-content-bignote" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`, `<ul>
<li class="task-list-item"><input type="checkbox" disabled="" data-source-position="3"/> If you want to rebase/retry this PR, click this checkbox.</li>
</ul>
<hr/>
<p>This PR has been generated by <a href="https://github.com/renovatebot/renovate" rel="nofollow">Renovate Bot</a>.</p>
`,
	}
}

// Test cases without ambiguous links
var sameCases = []string{
	// dear imgui wiki markdown extract: special wiki syntax
	`Wiki! Enjoy :)
- [[Links, Language bindings, Engine bindings|Links]]
- [[Tips]]

See commit 65f1bf27bc

Ideas and codes

- Bezier widget (by @r-lyeh) ` + AppURL + `ocornut/imgui/issues/786
- Bezier widget (by @r-lyeh) ` + AppURL + `gogits/gogs/issues/786
- Node graph editors https://github.com/ocornut/imgui/issues/306
- [[Memory Editor|memory_editor_example]]
- [[Plot var helper|plot_var_example]]`,
	// wine-staging wiki home extract: tables, special wiki syntax, images
	`## What is Wine Staging?
**Wine Staging** on website [wine-staging.com](http://wine-staging.com).

## Quick Links
Here are some links to the most important topics. You can find the full list of pages at the sidebar.

| [[images/icon-install.png]]    | [[Installation]]                                         |
|--------------------------------|----------------------------------------------------------|
| [[images/icon-usage.png]]      | [[Usage]]                                                |
`,
	// libgdx wiki page: inline images with special syntax
	`[Excelsior JET](http://www.excelsiorjet.com/) allows you to create native executables for Windows, Linux and Mac OS X.

1. [Package your libGDX application](https://github.com/libgdx/libgdx/wiki/Gradle-on-the-Commandline#packaging-for-the-desktop)
[[images/1.png]]
2. Perform a test run by hitting the Run! button.
[[images/2.png]]

## More tests {#custom-id}

(from https://www.markdownguide.org/extended-syntax/)

### Checkboxes

- [ ] unchecked
- [x] checked
- [ ] still unchecked

### Definition list

First Term
: This is the definition of the first term.

Second Term
: This is one definition of the second term.
: This is another definition of the second term.

### Footnotes

Here is a simple footnote,[^1] and here is a longer one.[^bignote]

[^1]: This is the first footnote.

[^bignote]: Here is one with multiple paragraphs and code.

    Indent paragraphs to include them in the footnote.

    ` + "`{ my code }`" + `

    Add as many paragraphs as you like.
`,
	`
- [ ] <!-- rebase-check --> If you want to rebase/retry this PR, click this checkbox.

---

This PR has been generated by [Renovate Bot](https://github.com/renovatebot/renovate).

<!-- test-comment -->`,
}

func TestTotal_RenderWiki(t *testing.T) {
	setting.AppURL = AppURL

	answers := testAnswers(util.URLJoin(FullURL, "wiki"), util.URLJoin(FullURL, "wiki", "raw"))

	for i := 0; i < len(sameCases); i++ {
		line, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: FullURL,
			},
			Metas:  localMetas,
			IsWiki: true,
		}, sameCases[i])
		require.NoError(t, err)
		assert.Equal(t, template.HTML(answers[i]), line)
	}

	testCases := []string{
		// Guard wiki sidebar: special syntax
		`[[Guardfile-DSL / Configuring-Guard|Guardfile-DSL---Configuring-Guard]]`,
		// rendered
		`<p><a href="` + FullURL + `wiki/Guardfile-DSL---Configuring-Guard" rel="nofollow">Guardfile-DSL / Configuring-Guard</a></p>
`,
		// special syntax
		`[[Name|Link]]`,
		// rendered
		`<p><a href="` + FullURL + `wiki/Link" rel="nofollow">Name</a></p>
`,
	}

	for i := 0; i < len(testCases); i += 2 {
		line, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: FullURL,
			},
			IsWiki: true,
		}, testCases[i])
		require.NoError(t, err)
		assert.Equal(t, template.HTML(testCases[i+1]), line)
	}
}

func TestTotal_RenderString(t *testing.T) {
	setting.AppURL = AppURL

	answers := testAnswers(util.URLJoin(FullURL, "src", "master"), util.URLJoin(FullURL, "media", "master"))

	for i := 0; i < len(sameCases); i++ {
		line, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base:       FullURL,
				BranchPath: "master",
			},
			Metas: localMetas,
		}, sameCases[i])
		require.NoError(t, err)
		assert.Equal(t, template.HTML(answers[i]), line)
	}

	testCases := []string{}

	for i := 0; i < len(testCases); i += 2 {
		line, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: FullURL,
			},
		}, testCases[i])
		require.NoError(t, err)
		assert.Equal(t, template.HTML(testCases[i+1]), line)
	}
}

func TestRender_RenderParagraphs(t *testing.T) {
	test := func(t *testing.T, str string, cnt int) {
		res, err := markdown.RenderRawString(&markup.RenderContext{Ctx: git.DefaultContext}, str)
		require.NoError(t, err)
		assert.Equal(t, cnt, strings.Count(res, "<p"), "Rendered result for unix should have %d paragraph(s) but has %d:\n%s\n", cnt, strings.Count(res, "<p"), res)

		mac := strings.ReplaceAll(str, "\n", "\r")
		res, err = markdown.RenderRawString(&markup.RenderContext{Ctx: git.DefaultContext}, mac)
		require.NoError(t, err)
		assert.Equal(t, cnt, strings.Count(res, "<p"), "Rendered result for mac should have %d paragraph(s) but has %d:\n%s\n", cnt, strings.Count(res, "<p"), res)

		dos := strings.ReplaceAll(str, "\n", "\r\n")
		res, err = markdown.RenderRawString(&markup.RenderContext{Ctx: git.DefaultContext}, dos)
		require.NoError(t, err)
		assert.Equal(t, cnt, strings.Count(res, "<p"), "Rendered result for windows should have %d paragraph(s) but has %d:\n%s\n", cnt, strings.Count(res, "<p"), res)
	}

	test(t, "\nOne\nTwo\nThree", 1)
	test(t, "\n\nOne\nTwo\nThree", 1)
	test(t, "\n\nOne\nTwo\nThree\n\n\n", 1)
	test(t, "A\n\nB\nC\n", 2)
	test(t, "A\n\n\nB\nC\n", 2)
}

func TestMarkdownRenderRaw(t *testing.T) {
	testcases := [][]byte{
		{ // clusterfuzz_testcase_minimized_fuzz_markdown_render_raw_6267570554535936
			0x2a, 0x20, 0x2d, 0x0a, 0x09, 0x20, 0x60, 0x5b, 0x0a, 0x09, 0x20, 0x60,
			0x5b,
		},
		{ // clusterfuzz_testcase_minimized_fuzz_markdown_render_raw_6278827345051648
			0x2d, 0x20, 0x2d, 0x0d, 0x09, 0x60, 0x0d, 0x09, 0x60,
		},
		{ // clusterfuzz_testcase_minimized_fuzz_markdown_render_raw_6016973788020736[] = {
			0x7b, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x3d, 0x35, 0x7d, 0x0a, 0x3d,
		},
	}

	for _, testcase := range testcases {
		log.Info("Test markdown render error with fuzzy data: %x, the following errors can be recovered", testcase)
		_, err := markdown.RenderRawString(&markup.RenderContext{Ctx: git.DefaultContext}, string(testcase))
		require.NoError(t, err)
	}
}

func TestRenderSiblingImages_Issue12925(t *testing.T) {
	testcase := `![image1](/image1)
![image2](/image2)
`
	expected := `<p><a href="/image1" target="_blank" rel="nofollow noopener"><img src="/image1" alt="image1"></a><br>
<a href="/image2" target="_blank" rel="nofollow noopener"><img src="/image2" alt="image2"></a></p>
`
	res, err := markdown.RenderRawString(&markup.RenderContext{Ctx: git.DefaultContext}, testcase)
	require.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestRenderEmojiInLinks_Issue12331(t *testing.T) {
	testcase := `[Link with emoji :moon: in text](https://gitea.io)`
	expected := `<p><a href="https://gitea.io" rel="nofollow">Link with emoji <span class="emoji" aria-label="waxing gibbous moon" data-alias="moon">🌔</span> in text</a></p>
`
	res, err := markdown.RenderString(&markup.RenderContext{Ctx: git.DefaultContext}, testcase)
	require.NoError(t, err)
	assert.Equal(t, template.HTML(expected), res)
}

func TestColorPreview(t *testing.T) {
	const nl = "\n"
	positiveTests := []struct {
		testcase string
		expected string
	}{
		{ // hex
			"`#FF0000`",
			`<p><code>#FF0000<span class="color-preview" style="background-color: #FF0000"></span></code></p>` + nl,
		},
		{ // rgb
			"`rgb(16, 32, 64)`",
			`<p><code>rgb(16, 32, 64)<span class="color-preview" style="background-color: rgb(16, 32, 64)"></span></code></p>` + nl,
		},
		{ // short hex
			"This is the color white `#000`",
			`<p>This is the color white <code>#000<span class="color-preview" style="background-color: #000"></span></code></p>` + nl,
		},
		{ // hsl
			"HSL stands for hue, saturation, and lightness. An example: `hsl(0, 100%, 50%)`.",
			`<p>HSL stands for hue, saturation, and lightness. An example: <code>hsl(0, 100%, 50%)<span class="color-preview" style="background-color: hsl(0, 100%, 50%)"></span></code>.</p>` + nl,
		},
		{ // uppercase hsl
			"HSL stands for hue, saturation, and lightness. An example: `HSL(0, 100%, 50%)`.",
			`<p>HSL stands for hue, saturation, and lightness. An example: <code>HSL(0, 100%, 50%)<span class="color-preview" style="background-color: HSL(0, 100%, 50%)"></span></code>.</p>` + nl,
		},
	}

	for _, test := range positiveTests {
		res, err := markdown.RenderString(&markup.RenderContext{Ctx: git.DefaultContext}, test.testcase)
		require.NoError(t, err, "Unexpected error in testcase: %q", test.testcase)
		assert.Equal(t, template.HTML(test.expected), res, "Unexpected result in testcase %q", test.testcase)
	}

	negativeTests := []string{
		// not a color code
		"`FF0000`",
		// inside a code block
		"```javascript" + nl + `const red = "#FF0000";` + nl + "```",
		// no backticks
		"rgb(166, 32, 64)",
		// typo
		"`hsI(0, 100%, 50%)`", // codespell:ignore
		// looks like a color but not really
		"`hsl(40, 60, 80)`",
	}

	for _, test := range negativeTests {
		res, err := markdown.RenderString(&markup.RenderContext{Ctx: git.DefaultContext}, test)
		require.NoError(t, err, "Unexpected error in testcase: %q", test)
		assert.NotContains(t, res, `<span class="color-preview" style="background-color: `, "Unexpected result in testcase %q", test)
	}
}

func TestMathBlock(t *testing.T) {
	const nl = "\n"
	testcases := []struct {
		testcase string
		expected string
	}{
		{
			"$a$",
			`<p><code class="language-math is-loading">a</code></p>` + nl,
		},
		{
			"$ a $",
			`<p><code class="language-math is-loading">a</code></p>` + nl,
		},
		{
			"$a$ $b$",
			`<p><code class="language-math is-loading">a</code> <code class="language-math is-loading">b</code></p>` + nl,
		},
		{
			`\(a\) \(b\)`,
			`<p><code class="language-math is-loading">a</code> <code class="language-math is-loading">b</code></p>` + nl,
		},
		{
			`$a$.`,
			`<p><code class="language-math is-loading">a</code>.</p>` + nl,
		},
		{
			`.$a$`,
			`<p>.$a$</p>` + nl,
		},
		{
			`$a a$b b$`,
			`<p>$a a$b b$</p>` + nl,
		},
		{
			`a a$b b`,
			`<p>a a$b b</p>` + nl,
		},
		{
			`a$b $a a$b b$`,
			`<p>a$b $a a$b b$</p>` + nl,
		},
		{
			"a$x$",
			`<p>a$x$</p>` + nl,
		},
		{
			"$x$a",
			`<p>$x$a</p>` + nl,
		},
		{
			"$$a$$",
			`<pre class="code-block is-loading"><code class="chroma language-math display">a</code></pre>` + nl,
		},
		{
			`\[a b\]`,
			`<pre class="code-block is-loading"><code class="chroma language-math display">a b</code></pre>` + nl,
		},
		{
			`\[a b]`,
			`<p>[a b]</p>` + nl,
		},
		{
			`$$a`,
			`<p>$$a</p>` + nl,
		},
		{
			"$a$ ($b$) [$c$] {$d$}",
			`<p><code class="language-math is-loading">a</code> (<code class="language-math is-loading">b</code>) [$c$] {$d$}</p>` + nl,
		},
		{
			"$$a$$ test",
			`<p><code class="language-math display is-loading">a</code> test</p>` + nl,
		},
		{
			"test $$a$$",
			`<p>test <code class="language-math display is-loading">a</code></p>` + nl,
		},
	}

	for _, test := range testcases {
		res, err := markdown.RenderString(&markup.RenderContext{Ctx: git.DefaultContext}, test.testcase)
		require.NoError(t, err, "Unexpected error in testcase: %q", test.testcase)
		assert.Equal(t, template.HTML(test.expected), res, "Unexpected result in testcase %q", test.testcase)
	}
}

func TestFootnote(t *testing.T) {
	testcases := []struct {
		testcase string
		expected string
	}{
		{
			`Citation needed[^0].
[^0]: Source`,
			`<p>Citation needed<sup id="fnref:user-content-0"><a href="#fn:user-content-0" rel="nofollow">1</a></sup>.</p>
<div>
<hr/>
<ol>
<li id="fn:user-content-0">
<p>Source <a href="#fnref:user-content-0" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Citation needed[^0]`,
			`<p>Citation needed[^0]</p>
`,
		},
		{
			`Citation needed[^1], Citation needed twice[^3]
[^3]: Source`,
			`<p>Citation needed[^1], Citation needed twice<sup id="fnref:user-content-3"><a href="#fn:user-content-3" rel="nofollow">1</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-3">
<p>Source <a href="#fnref:user-content-3" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Citation needed[^0]
[^1]: Source`,
			`<p>Citation needed[^0]</p>
`,
		},
		{
			`Citation needed[^0]
[^0]: Source 1
[^0]: Source 2`,
			`<p>Citation needed<sup id="fnref:user-content-0"><a href="#fn:user-content-0" rel="nofollow">1</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-0">
<p>Source 1 <a href="#fnref:user-content-0" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Citation needed![^0]
[^0]: Source`,
			`<p>Citation needed<sup id="fnref:user-content-0"><a href="#fn:user-content-0" rel="nofollow">1</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-0">
<p>Source <a href="#fnref:user-content-0" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Trigger [^`,
			`<p>Trigger [^</p>
`,
		},
		{
			`Trigger 2 [^0`,
			`<p>Trigger 2 [^0</p>
`,
		},
		{
			`Citation needed[^0]
[^0]: Source with citation needed[^1]
[^1]: Source`,
			`<p>Citation needed<sup id="fnref:user-content-0"><a href="#fn:user-content-0" rel="nofollow">1</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-0">
<p>Source with citation needed<sup id="fnref:user-content-1"><a href="#fn:user-content-1" rel="nofollow">2</a></sup> <a href="#fnref:user-content-0" rel="nofollow">↩︎</a></p>
</li>
<li id="fn:user-content-1">
<p>Source <a href="#fnref:user-content-1" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Citation needed[^#]
[^#]: Source`,
			`<p>Citation needed<sup id="fnref:user-content-1"><a href="#fn:user-content-1" rel="nofollow">1</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-1">
<p>Source <a href="#fnref:user-content-1" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Citation needed[^0]
    [^0]: Source`,
			`<p>Citation needed[^0]<br/>
[^0]: Source</p>
`,
		},
		{
			`[^0]: Source

Citation needed[^0].`,
			`<p>Citation needed<sup id="fnref:user-content-0"><a href="#fn:user-content-0" rel="nofollow">1</a></sup>.</p>
<div>
<hr/>
<ol>
<li id="fn:user-content-0">
<p>Source <a href="#fnref:user-content-0" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Citation needed[^]
[^]: Source`,
			`<p>Citation needed[^]<br/>
[^]: Source</p>
`,
		},
		{
			`Citation needed[^0]
[^0] Source`,
			`<p>Citation needed[^0]<br/>
[^0] Source</p>
`,
		},
		{
			`Citation needed[^0]
[^0 Source`,
			`<p>Citation needed[^0]<br/>
[^0 Source</p>
`,
		},
		{
			`Citation needed[^0] [^0]: Source`,
			`<p>Citation needed[^0] [^0]: Source</p>
`,
		},
		{
			`Citation needed[^Source here 0 # 9-3]
[^Source here 0 # 9-3]: Source`,
			`<p>Citation needed<sup id="fnref:user-content-source-here-0-9-3"><a href="#fn:user-content-source-here-0-9-3" rel="nofollow">1</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-source-here-0-9-3">
<p>Source <a href="#fnref:user-content-source-here-0-9-3" rel="nofollow">↩︎</a></p>
</li>
</ol>
</div>
`,
		},
		{
			`Citation needed[^0]
[^0]:`,
			`<p>Citation needed<sup id="fnref:user-content-0"><a href="#fn:user-content-0" rel="nofollow">1</a></sup></p>
<div>
<hr/>
<ol>
<li id="fn:user-content-0">
 <a href="#fnref:user-content-0" rel="nofollow">↩︎</a></li>
</ol>
</div>
`,
		},
	}
	for _, test := range testcases {
		res, err := markdown.RenderString(&markup.RenderContext{Ctx: git.DefaultContext}, test.testcase)
		require.NoError(t, err, "Unexpected error in testcase: %q", test.testcase)
		assert.Equal(t, test.expected, string(res), "Unexpected result in testcase %q", test.testcase)
	}
}

func TestTaskList(t *testing.T) {
	testcases := []struct {
		testcase string
		expected string
	}{
		{
			// data-source-position should take into account YAML frontmatter.
			`---
foo: bar
---
- [ ] task 1`,
			`<details><summary><i class="icon table"></i></summary><table>
<thead>
<tr>
<th>foo</th>
</tr>
</thead>
<tbody>
<tr>
<td>bar</td>
</tr>
</tbody>
</table>
</details><ul>
<li class="task-list-item"><input type="checkbox" disabled="" data-source-position="19"/>task 1</li>
</ul>
`,
		},
	}

	for _, test := range testcases {
		res, err := markdown.RenderString(&markup.RenderContext{Ctx: git.DefaultContext}, test.testcase)
		require.NoError(t, err, "Unexpected error in testcase: %q", test.testcase)
		assert.Equal(t, template.HTML(test.expected), res, "Unexpected result in testcase %q", test.testcase)
	}
}

func TestRenderLinks(t *testing.T) {
	input := `  space @mention-user${SPACE}${SPACE}
/just/a/path.bin
https://example.com/file.bin
[local link](file.bin)
[remote link](https://example.com)
[[local link|file.bin]]
[[remote link|https://example.com]]
![local image](image.jpg)
![local image](path/file)
![local image](/path/file)
![remote image](https://example.com/image.jpg)
[[local image|image.jpg]]
[[remote link|https://example.com/image.jpg]]
https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare
https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit
:+1:
mail@domain.com
@mention-user test
#123
  space${SPACE}${SPACE}
`
	input = strings.ReplaceAll(input, "${SPACE}", " ") // replace ${SPACE} with " ", to avoid some editor's auto-trimming
	cases := []struct {
		Links    markup.Links
		IsWiki   bool
		Expected string
	}{
		{ // 0
			Links:  markup.Links{},
			IsWiki: false,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/src/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/image.jpg" target="_blank" rel="nofollow noopener"><img src="/image.jpg" alt="local image"/></a><br/>
<a href="/path/file" target="_blank" rel="nofollow noopener"><img src="/path/file" alt="local image"/></a><br/>
<a href="/path/file" target="_blank" rel="nofollow noopener"><img src="/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/image.jpg" rel="nofollow"><img src="/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 1
			Links:  markup.Links{},
			IsWiki: true,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/wiki/raw/image.jpg" target="_blank" rel="nofollow noopener"><img src="/wiki/raw/image.jpg" alt="local image"/></a><br/>
<a href="/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/wiki/raw/image.jpg" rel="nofollow"><img src="/wiki/raw/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 2
			Links: markup.Links{
				Base: "https://gitea.io/",
			},
			IsWiki: false,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="https://gitea.io/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="https://gitea.io/src/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="https://gitea.io/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://gitea.io/image.jpg" alt="local image"/></a><br/>
<a href="https://gitea.io/path/file" target="_blank" rel="nofollow noopener"><img src="https://gitea.io/path/file" alt="local image"/></a><br/>
<a href="https://gitea.io/path/file" target="_blank" rel="nofollow noopener"><img src="https://gitea.io/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="https://gitea.io/image.jpg" rel="nofollow"><img src="https://gitea.io/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 3
			Links: markup.Links{
				Base: "https://gitea.io/",
			},
			IsWiki: true,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="https://gitea.io/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="https://gitea.io/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="https://gitea.io/wiki/raw/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://gitea.io/wiki/raw/image.jpg" alt="local image"/></a><br/>
<a href="https://gitea.io/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="https://gitea.io/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="https://gitea.io/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="https://gitea.io/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="https://gitea.io/wiki/raw/image.jpg" rel="nofollow"><img src="https://gitea.io/wiki/raw/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 4
			Links: markup.Links{
				Base: "/relative/path",
			},
			IsWiki: false,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/relative/path/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/src/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/image.jpg" target="_blank" rel="nofollow noopener"><img src="/relative/path/image.jpg" alt="local image"/></a><br/>
<a href="/relative/path/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/path/file" alt="local image"/></a><br/>
<a href="/relative/path/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/relative/path/image.jpg" rel="nofollow"><img src="/relative/path/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 5
			Links: markup.Links{
				Base: "/relative/path",
			},
			IsWiki: true,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/raw/image.jpg" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/image.jpg" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/relative/path/wiki/raw/image.jpg" rel="nofollow"><img src="/relative/path/wiki/raw/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 6
			Links: markup.Links{
				Base:       "/user/repo",
				BranchPath: "branch/main",
			},
			IsWiki: false,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/user/repo/src/branch/main/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/user/repo/src/branch/main/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/user/repo/media/branch/main/image.jpg" target="_blank" rel="nofollow noopener"><img src="/user/repo/media/branch/main/image.jpg" alt="local image"/></a><br/>
<a href="/user/repo/media/branch/main/path/file" target="_blank" rel="nofollow noopener"><img src="/user/repo/media/branch/main/path/file" alt="local image"/></a><br/>
<a href="/user/repo/media/branch/main/path/file" target="_blank" rel="nofollow noopener"><img src="/user/repo/media/branch/main/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/user/repo/media/branch/main/image.jpg" rel="nofollow"><img src="/user/repo/media/branch/main/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 7
			Links: markup.Links{
				Base:       "/relative/path",
				BranchPath: "branch/main",
			},
			IsWiki: true,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/raw/image.jpg" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/image.jpg" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/relative/path/wiki/raw/image.jpg" rel="nofollow"><img src="/relative/path/wiki/raw/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 8
			Links: markup.Links{
				Base:     "/user/repo",
				TreePath: "sub/folder",
			},
			IsWiki: false,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/user/repo/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/user/repo/src/sub/folder/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/user/repo/image.jpg" target="_blank" rel="nofollow noopener"><img src="/user/repo/image.jpg" alt="local image"/></a><br/>
<a href="/user/repo/path/file" target="_blank" rel="nofollow noopener"><img src="/user/repo/path/file" alt="local image"/></a><br/>
<a href="/user/repo/path/file" target="_blank" rel="nofollow noopener"><img src="/user/repo/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/user/repo/image.jpg" rel="nofollow"><img src="/user/repo/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 9
			Links: markup.Links{
				Base:     "/relative/path",
				TreePath: "sub/folder",
			},
			IsWiki: true,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/raw/image.jpg" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/image.jpg" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/relative/path/wiki/raw/image.jpg" rel="nofollow"><img src="/relative/path/wiki/raw/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 10
			Links: markup.Links{
				Base:       "/user/repo",
				BranchPath: "branch/main",
				TreePath:   "sub/folder",
			},
			IsWiki: false,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/user/repo/src/branch/main/sub/folder/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/user/repo/src/branch/main/sub/folder/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/user/repo/media/branch/main/sub/folder/image.jpg" target="_blank" rel="nofollow noopener"><img src="/user/repo/media/branch/main/sub/folder/image.jpg" alt="local image"/></a><br/>
<a href="/user/repo/media/branch/main/sub/folder/path/file" target="_blank" rel="nofollow noopener"><img src="/user/repo/media/branch/main/sub/folder/path/file" alt="local image"/></a><br/>
<a href="/user/repo/media/branch/main/sub/folder/path/file" target="_blank" rel="nofollow noopener"><img src="/user/repo/media/branch/main/sub/folder/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/user/repo/media/branch/main/sub/folder/image.jpg" rel="nofollow"><img src="/user/repo/media/branch/main/sub/folder/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
		{ // 11
			Links: markup.Links{
				Base:       "/relative/path",
				BranchPath: "branch/main",
				TreePath:   "sub/folder",
			},
			IsWiki: true,
			Expected: `<p>space @mention-user<br/>
/just/a/path.bin<br/>
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/file.bin" rel="nofollow">local link</a><br/>
<a href="https://example.com" rel="nofollow">remote link</a><br/>
<a href="/relative/path/wiki/raw/image.jpg" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/image.jpg" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="/relative/path/wiki/raw/path/file" target="_blank" rel="nofollow noopener"><img src="/relative/path/wiki/raw/path/file" alt="local image"/></a><br/>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a><br/>
<a href="/relative/path/wiki/raw/image.jpg" rel="nofollow"><img src="/relative/path/wiki/raw/image.jpg" title="local image" alt=""/></a><br/>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a><br/>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow">https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare<br/>
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow">https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb</a><br/>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit<br/>
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><br/>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a><br/>
@mention-user test<br/>
#123<br/>
space</p>
`,
		},
	}

	for i, c := range cases {
		result, err := markdown.RenderString(&markup.RenderContext{Ctx: t.Context(), Links: c.Links, IsWiki: c.IsWiki}, input)
		require.NoError(t, err, "Unexpected error in testcase: %v", i)
		assert.Equal(t, template.HTML(c.Expected), result, "Unexpected result in testcase %v", i)
	}
}

func TestCustomMarkdownURL(t *testing.T) {
	defer test.MockVariableValue(&setting.Markdown.CustomURLSchemes, []string{"abp"})()
	setting.AppURL = AppURL

	test := func(input, expected string) {
		buffer, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base:       FullURL,
				BranchPath: "branch/main",
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(buffer)))
	}

	test("[test](abp:subscribe?location=https://codeberg.org/filters.txt&amp;title=joy)",
		`<p><a href="abp:subscribe?location=https://codeberg.org/filters.txt&amp;title=joy" rel="nofollow">test</a></p>`)

	// Ensure that the schema itself without `:` is still made absolute.
	test("[test](abp)",
		`<p><a href="http://localhost:3000/gogits/gogs/src/branch/main/abp" rel="nofollow">test</a></p>`)
}

func TestYAMLMeta(t *testing.T) {
	setting.AppURL = AppURL

	test := func(input, expected string) {
		buffer, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(buffer)))
	}

	test(`---
include_toc: true
---
## Header`,
		`<details><summary><i class="icon table"></i></summary><table>
<thead>
<tr>
<th>include_toc</th>
</tr>
</thead>
<tbody>
<tr>
<td>true</td>
</tr>
</tbody>
</table>
</details><details><summary>toc</summary><ul>
<li>
<a href="#user-content-header" rel="nofollow">Header</a></li>
</ul>
</details><h2 id="user-content-header">Header</h2>`)

	test(`---
key: value
---`,
		`<details><summary><i class="icon table"></i></summary><table>
<thead>
<tr>
<th>key</th>
</tr>
</thead>
<tbody>
<tr>
<td>value</td>
</tr>
</tbody>
</table>
</details>`)

	test("---\n---\n",
		`<hr/>
<hr/>`)

	test(`---
gitea:
  details_icon: smiley
  include_toc: true
---
# Another header`,
		`<details><summary><i class="icon smiley"></i></summary><table>
<thead>
<tr>
<th>gitea</th>
</tr>
</thead>
<tbody>
<tr>
<td><table>
<thead>
<tr>
<th>details_icon</th>
<th>include_toc</th>
</tr>
</thead>
<tbody>
<tr>
<td>smiley</td>
<td>true</td>
</tr>
</tbody>
</table>
</td>
</tr>
</tbody>
</table>
</details><details><summary>toc</summary><ul>
<li>
<a href="#user-content-another-header" rel="nofollow">Another header</a></li>
</ul>
</details><h1 id="user-content-another-header">Another header</h1>`)

	test(`---
gitea:
  meta: table
key: value
---`, `<table>
<thead>
<tr>
<th>gitea</th>
<th>key</th>
</tr>
</thead>
<tbody>
<tr>
<td><table>
<thead>
<tr>
<th>meta</th>
</tr>
</thead>
<tbody>
<tr>
<td>table</td>
</tr>
</tbody>
</table>
</td>
<td>value</td>
</tr>
</tbody>
</table>`)
}

func TestCallout(t *testing.T) {
	setting.AppURL = AppURL

	test := func(input, expected string) {
		buffer, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(buffer)))
	}

	test(">\n0", "<blockquote>\n</blockquote>\n<p>0</p>")
	test("> **Warning**\n> Bad stuff is brewing here", `<blockquote class="attention-header attention-warning"><p class="attention-title"><strong class="attention-warning">Warning</strong></p>
<p>Bad stuff is brewing here</p>
</blockquote>`)
	test("> [!WARNING]\n> Bad stuff is brewing here", `<blockquote class="attention-header attention-warning"><p class="attention-title"><strong class="attention-warning">Warning</strong></p>
<p>Bad stuff is brewing here</p>
</blockquote>`)
}
