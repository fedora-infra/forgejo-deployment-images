// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package markup_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/emoji"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/translation"
	"forgejo.org/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var localMetas = map[string]string{
	"user":     "gogits",
	"repo":     "gogs",
	"repoPath": "../../tests/gitea-repositories-meta/user13/repo11.git/",
}

func TestMain(m *testing.M) {
	unittest.InitSettings()
	if err := git.InitSimple(context.Background()); err != nil {
		log.Fatal("git init failed, err: %v", err)
	}
	os.Exit(m.Run())
}

func TestRender_Commits(t *testing.T) {
	setting.AppURL = markup.TestAppURL
	test := func(input, expected string) {
		buffer, err := markup.RenderString(&markup.RenderContext{
			Ctx:          git.DefaultContext,
			RelativePath: ".md",
			Links: markup.Links{
				AbsolutePrefix: true,
				Base:           markup.TestRepoURL,
			},
			Metas: localMetas,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	sha := "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	repo := markup.TestRepoURL
	commit := util.URLJoin(repo, "commit", sha)
	tree := util.URLJoin(repo, "tree", sha, "src")

	file := util.URLJoin(repo, "commit", sha, "example.txt")
	fileWithExtra := file + ":"
	fileWithHash := file + "#L2"
	fileWithHasExtra := file + "#L2:"
	commitCompare := util.URLJoin(repo, "compare", sha+"..."+sha)
	commitCompareWithHash := commitCompare + "#L2"

	test(sha, `<p><a href="`+commit+`" rel="nofollow"><code>65f1bf27bc</code></a></p>`)
	test(sha[:7], `<p><a href="`+commit[:len(commit)-(40-7)]+`" rel="nofollow"><code>65f1bf2</code></a></p>`)
	test(sha[:39], `<p><a href="`+commit[:len(commit)-(40-39)]+`" rel="nofollow"><code>65f1bf27bc</code></a></p>`)
	test(commit, `<p><a href="`+commit+`" rel="nofollow"><code>65f1bf27bc</code></a></p>`)
	test(tree, `<p><a href="`+tree+`" rel="nofollow"><code>65f1bf27bc/src</code></a></p>`)

	test(file, `<p><a href="`+file+`" rel="nofollow"><code>65f1bf27bc/example.txt</code></a></p>`)
	test(fileWithExtra, `<p><a href="`+file+`" rel="nofollow"><code>65f1bf27bc/example.txt</code></a>:</p>`)
	test(fileWithHash, `<p><a href="`+fileWithHash+`" rel="nofollow"><code>65f1bf27bc/example.txt (L2)</code></a></p>`)
	test(fileWithHasExtra, `<p><a href="`+fileWithHash+`" rel="nofollow"><code>65f1bf27bc/example.txt (L2)</code></a>:</p>`)
	test(commitCompare, `<p><a href="`+commitCompare+`" rel="nofollow"><code>65f1bf27bc...65f1bf27bc</code></a></p>`)
	test(commitCompareWithHash, `<p><a href="`+commitCompareWithHash+`" rel="nofollow"><code>65f1bf27bc...65f1bf27bc (L2)</code></a></p>`)

	test("commit "+sha, `<p>commit <a href="`+commit+`" rel="nofollow"><code>65f1bf27bc</code></a></p>`)
	test("/home/gitea/"+sha, "<p>/home/gitea/"+sha+"</p>")
	test("deadbeef", `<p>deadbeef</p>`)
	test("d27ace93", `<p>d27ace93</p>`)
	test(sha[:14]+".x", `<p>`+sha[:14]+`.x</p>`)

	expected14 := `<a href="` + commit[:len(commit)-(40-14)] + `" rel="nofollow"><code>` + sha[:10] + `</code></a>`
	test(sha[:14]+".", `<p>`+expected14+`.</p>`)
	test(sha[:14]+",", `<p>`+expected14+`,</p>`)
	test("["+sha[:14]+"]", `<p>[`+expected14+`]</p>`)
}

func TestRender_CrossReferences(t *testing.T) {
	setting.AppURL = markup.TestAppURL

	test := func(input, expected string) {
		buffer, err := markup.RenderString(&markup.RenderContext{
			Ctx:          git.DefaultContext,
			RelativePath: "a.md",
			Links: markup.Links{
				AbsolutePrefix: true,
				Base:           setting.AppSubURL,
			},
			Metas: localMetas,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	test(
		"gogits/gogs#12345",
		`<p><a href="`+util.URLJoin(markup.TestAppURL, "gogits", "gogs", "issues", "12345")+`" class="ref-issue" rel="nofollow">gogits/gogs#12345</a></p>`)
	test(
		"go-gitea/gitea#12345",
		`<p><a href="`+util.URLJoin(markup.TestAppURL, "go-gitea", "gitea", "issues", "12345")+`" class="ref-issue" rel="nofollow">go-gitea/gitea#12345</a></p>`)
	test(
		"/home/gitea/go-gitea/gitea#12345",
		`<p>/home/gitea/go-gitea/gitea#12345</p>`)
	test(
		util.URLJoin(markup.TestAppURL, "gogitea", "gitea", "issues", "12345"),
		`<p><a href="`+util.URLJoin(markup.TestAppURL, "gogitea", "gitea", "issues", "12345")+`" class="ref-issue" rel="nofollow">gogitea/gitea#12345</a></p>`)
	test(
		util.URLJoin(markup.TestAppURL, "go-gitea", "gitea", "issues", "12345"),
		`<p><a href="`+util.URLJoin(markup.TestAppURL, "go-gitea", "gitea", "issues", "12345")+`" class="ref-issue" rel="nofollow">go-gitea/gitea#12345</a></p>`)
	test(
		util.URLJoin(markup.TestAppURL, "gogitea", "some-repo-name", "issues", "12345"),
		`<p><a href="`+util.URLJoin(markup.TestAppURL, "gogitea", "some-repo-name", "issues", "12345")+`" class="ref-issue" rel="nofollow">gogitea/some-repo-name#12345</a></p>`)

	sha := "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	urlWithQuery := util.URLJoin(markup.TestAppURL, "forgejo", "some-repo-name", "commit", sha, "README.md") + "?display=source#L1-L5"
	test(
		urlWithQuery,
		`<p><a href="`+urlWithQuery+`" rel="nofollow"><code>`+sha[:10]+`/README.md (L1-L5)</code></a></p>`)
}

func TestRender_links(t *testing.T) {
	setting.AppURL = markup.TestAppURL

	test := func(input, expected string) {
		buffer, err := markup.RenderString(&markup.RenderContext{
			Ctx:          git.DefaultContext,
			RelativePath: "a.md",
			Links: markup.Links{
				Base: markup.TestRepoURL,
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}
	// Text that should be turned into URL

	defaultCustom := setting.Markdown.CustomURLSchemes
	setting.Markdown.CustomURLSchemes = []string{"ftp", "magnet"}
	markup.InitializeSanitizer()
	markup.CustomLinkURLSchemes(setting.Markdown.CustomURLSchemes)

	test(
		"https://www.example.com",
		`<p><a href="https://www.example.com" rel="nofollow">https://www.example.com</a></p>`)
	test(
		"http://www.example.com",
		`<p><a href="http://www.example.com" rel="nofollow">http://www.example.com</a></p>`)
	test(
		"https://example.com",
		`<p><a href="https://example.com" rel="nofollow">https://example.com</a></p>`)
	test(
		"http://example.com",
		`<p><a href="http://example.com" rel="nofollow">http://example.com</a></p>`)
	test(
		"http://foo.com/blah_blah",
		`<p><a href="http://foo.com/blah_blah" rel="nofollow">http://foo.com/blah_blah</a></p>`)
	test(
		"http://foo.com/blah_blah/",
		`<p><a href="http://foo.com/blah_blah/" rel="nofollow">http://foo.com/blah_blah/</a></p>`)
	test(
		"http://www.example.com/wpstyle/?p=364",
		`<p><a href="http://www.example.com/wpstyle/?p=364" rel="nofollow">http://www.example.com/wpstyle/?p=364</a></p>`)
	test(
		"https://www.example.com/foo/?bar=baz&inga=42&quux",
		`<p><a href="https://www.example.com/foo/?bar=baz&amp;inga=42&amp;quux" rel="nofollow">https://www.example.com/foo/?bar=baz&amp;inga=42&amp;quux</a></p>`)
	test(
		"http://142.42.1.1/",
		`<p><a href="http://142.42.1.1/" rel="nofollow">http://142.42.1.1/</a></p>`)
	test(
		"https://github.com/go-gitea/gitea/?p=aaa/bbb.html#ccc-ddd",
		`<p><a href="https://github.com/go-gitea/gitea/?p=aaa/bbb.html#ccc-ddd" rel="nofollow">https://github.com/go-gitea/gitea/?p=aaa/bbb.html#ccc-ddd</a></p>`)
	test(
		"https://en.wikipedia.org/wiki/URL_(disambiguation)",
		`<p><a href="https://en.wikipedia.org/wiki/URL_(disambiguation)" rel="nofollow">https://en.wikipedia.org/wiki/URL_(disambiguation)</a></p>`)
	test(
		"https://foo_bar.example.com/",
		`<p><a href="https://foo_bar.example.com/" rel="nofollow">https://foo_bar.example.com/</a></p>`)
	test(
		"https://stackoverflow.com/questions/2896191/what-is-go-used-fore",
		`<p><a href="https://stackoverflow.com/questions/2896191/what-is-go-used-fore" rel="nofollow">https://stackoverflow.com/questions/2896191/what-is-go-used-fore</a></p>`)
	test(
		"https://username:password@gitea.com",
		`<p><a href="https://username:password@gitea.com" rel="nofollow">https://username:password@gitea.com</a></p>`)
	test(
		"ftp://gitea.com/file.txt",
		`<p><a href="ftp://gitea.com/file.txt" rel="nofollow">ftp://gitea.com/file.txt</a></p>`)
	test(
		"magnet:?xt=urn:btih:5dee65101db281ac9c46344cd6b175cdcadabcde&dn=download",
		`<p><a href="magnet:?xt=urn:btih:5dee65101db281ac9c46344cd6b175cdcadabcde&amp;dn=download" rel="nofollow">magnet:?xt=urn:btih:5dee65101db281ac9c46344cd6b175cdcadabcde&amp;dn=download</a></p>`)

	// Test that should *not* be turned into URL
	test(
		"www.example.com",
		`<p>www.example.com</p>`)
	test(
		"example.com",
		`<p>example.com</p>`)
	test(
		"test.example.com",
		`<p>test.example.com</p>`)
	test(
		"http://",
		`<p>http://</p>`)
	test(
		"https://",
		`<p>https://</p>`)
	test(
		"://",
		`<p>://</p>`)
	test(
		"www",
		`<p>www</p>`)
	test(
		"ftps://gitea.com",
		`<p>ftps://gitea.com</p>`)

	// Restore previous settings
	setting.Markdown.CustomURLSchemes = defaultCustom
	markup.InitializeSanitizer()
	markup.CustomLinkURLSchemes(setting.Markdown.CustomURLSchemes)
}

func TestRender_email(t *testing.T) {
	setting.AppURL = markup.TestAppURL

	test := func(input, expected string) {
		res, err := markup.RenderString(&markup.RenderContext{
			Ctx:          git.DefaultContext,
			RelativePath: "a.md",
			Links: markup.Links{
				Base: markup.TestRepoURL,
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(res))
	}
	// Text that should be turned into email link

	test(
		"info@gitea.com",
		`<p><a href="mailto:info@gitea.com" rel="nofollow">info@gitea.com</a></p>`)
	test(
		"(info@gitea.com)",
		`<p>(<a href="mailto:info@gitea.com" rel="nofollow">info@gitea.com</a>)</p>`)
	test(
		"[info@gitea.com]",
		`<p>[<a href="mailto:info@gitea.com" rel="nofollow">info@gitea.com</a>]</p>`)
	test(
		"info@gitea.com.",
		`<p><a href="mailto:info@gitea.com" rel="nofollow">info@gitea.com</a>.</p>`)
	test(
		"firstname+lastname@gitea.com",
		`<p><a href="mailto:firstname+lastname@gitea.com" rel="nofollow">firstname+lastname@gitea.com</a></p>`)
	test(
		"send email to info@gitea.co.uk.",
		`<p>send email to <a href="mailto:info@gitea.co.uk" rel="nofollow">info@gitea.co.uk</a>.</p>`)

	test(
		`j.doe@example.com,
	j.doe@example.com.
	j.doe@example.com;
	j.doe@example.com?
	j.doe@example.com!`,
		`<p><a href="mailto:j.doe@example.com" rel="nofollow">j.doe@example.com</a>,<br/>
<a href="mailto:j.doe@example.com" rel="nofollow">j.doe@example.com</a>.<br/>
<a href="mailto:j.doe@example.com" rel="nofollow">j.doe@example.com</a>;<br/>
<a href="mailto:j.doe@example.com" rel="nofollow">j.doe@example.com</a>?<br/>
<a href="mailto:j.doe@example.com" rel="nofollow">j.doe@example.com</a>!</p>`)

	// Test that should *not* be turned into email links
	test(
		"\"info@gitea.com\"",
		`<p>&#34;info@gitea.com&#34;</p>`)
	test(
		"/home/gitea/mailstore/info@gitea/com",
		`<p>/home/gitea/mailstore/info@gitea/com</p>`)
	test(
		"git@try.gitea.io:go-gitea/gitea.git",
		`<p>git@try.gitea.io:go-gitea/gitea.git</p>`)
	test(
		"gitea@3",
		`<p>gitea@3</p>`)
	test(
		"gitea@gmail.c",
		`<p>gitea@gmail.c</p>`)
	test(
		"email@domain@domain.com",
		`<p>email@domain@domain.com</p>`)
	test(
		"email@domain..com",
		`<p>email@domain..com</p>`)
}

func TestRender_emoji(t *testing.T) {
	setting.AppURL = markup.TestAppURL
	setting.StaticURLPrefix = markup.TestAppURL

	test := func(input, expected string) {
		expected = strings.ReplaceAll(expected, "&", "&amp;")
		buffer, err := markup.RenderString(&markup.RenderContext{
			Ctx:          git.DefaultContext,
			RelativePath: "a.md",
			Links: markup.Links{
				Base: markup.TestRepoURL,
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	// Make sure we can successfully match every emoji in our dataset with regex
	for i := range emoji.GemojiData {
		test(
			emoji.GemojiData[i].Emoji,
			`<p><span class="emoji" aria-label="`+emoji.GemojiData[i].Description+`" data-alias="`+emoji.GemojiData[i].Aliases[0]+`">`+emoji.GemojiData[i].Emoji+`</span></p>`)
	}
	for i := range emoji.GemojiData {
		test(
			":"+emoji.GemojiData[i].Aliases[0]+":",
			`<p><span class="emoji" aria-label="`+emoji.GemojiData[i].Description+`" data-alias="`+emoji.GemojiData[i].Aliases[0]+`">`+emoji.GemojiData[i].Emoji+`</span></p>`)
	}

	// Text that should be turned into or recognized as emoji
	test(
		":gitea:",
		`<p><span class="emoji" aria-label="gitea" data-alias="gitea"><img alt=":gitea:" src="`+setting.StaticURLPrefix+`/assets/img/emoji/gitea.png"/></span></p>`)
	test(
		":custom-emoji:",
		`<p>:custom-emoji:</p>`)
	setting.UI.CustomEmojisMap["custom-emoji"] = ":custom-emoji:"
	test(
		":custom-emoji:",
		`<p><span class="emoji" aria-label="custom-emoji" data-alias="custom-emoji"><img alt=":custom-emoji:" src="`+setting.StaticURLPrefix+`/assets/img/emoji/custom-emoji.png"/></span></p>`)
	test(
		"这是字符:1::+1: some🐊 \U0001f44d:custom-emoji: :gitea:",
		`<p>这是字符:1:<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span> some<span class="emoji" aria-label="crocodile" data-alias="crocodile">🐊</span> `+
			`<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span><span class="emoji" aria-label="custom-emoji" data-alias="custom-emoji"><img alt=":custom-emoji:" src="`+setting.StaticURLPrefix+`/assets/img/emoji/custom-emoji.png"/></span> `+
			`<span class="emoji" aria-label="gitea" data-alias="gitea"><img alt=":gitea:" src="`+setting.StaticURLPrefix+`/assets/img/emoji/gitea.png"/></span></p>`)
	test(
		"Some text with 😄 in the middle",
		`<p>Some text with <span class="emoji" aria-label="grinning face with smiling eyes" data-alias="smile">😄</span> in the middle</p>`)
	test(
		"Some text with :smile: in the middle",
		`<p>Some text with <span class="emoji" aria-label="grinning face with smiling eyes" data-alias="smile">😄</span> in the middle</p>`)
	test(
		"Some text with 😄😄 2 emoji next to each other",
		`<p>Some text with <span class="emoji" aria-label="grinning face with smiling eyes" data-alias="smile">😄</span><span class="emoji" aria-label="grinning face with smiling eyes" data-alias="smile">😄</span> 2 emoji next to each other</p>`)
	test(
		"😎🤪🔐🤑❓",
		`<p><span class="emoji" aria-label="smiling face with sunglasses" data-alias="sunglasses">😎</span><span class="emoji" aria-label="zany face" data-alias="zany_face">🤪</span><span class="emoji" aria-label="locked with key" data-alias="closed_lock_with_key">🔐</span><span class="emoji" aria-label="money-mouth face" data-alias="money_mouth_face">🤑</span><span class="emoji" aria-label="red question mark" data-alias="question">❓</span></p>`)

	// should match nothing
	test(
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		`<p>2001:0db8:85a3:0000:0000:8a2e:0370:7334</p>`)
	test(
		":not exist:",
		`<p>:not exist:</p>`)
}

func TestRender_ShortLinks(t *testing.T) {
	setting.AppURL = markup.TestAppURL
	tree := util.URLJoin(markup.TestRepoURL, "src", "master")

	test := func(input, expected, expectedWiki string) {
		buffer, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base:       markup.TestRepoURL,
				BranchPath: "master",
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(buffer)))
		buffer, err = markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: markup.TestRepoURL,
			},
			Metas:  localMetas,
			IsWiki: true,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expectedWiki), strings.TrimSpace(string(buffer)))
	}

	mediatree := util.URLJoin(markup.TestRepoURL, "media", "master")
	url := util.URLJoin(tree, "Link")
	otherURL := util.URLJoin(tree, "Other-Link")
	encodedURL := util.URLJoin(tree, "Link%3F")
	imgurl := util.URLJoin(mediatree, "Link.jpg")
	otherImgurl := util.URLJoin(mediatree, "Link+Other.jpg")
	encodedImgurl := util.URLJoin(mediatree, "Link+%23.jpg")
	notencodedImgurl := util.URLJoin(mediatree, "some", "path", "Link+#.jpg")
	urlWiki := util.URLJoin(markup.TestRepoURL, "wiki", "Link")
	otherURLWiki := util.URLJoin(markup.TestRepoURL, "wiki", "Other-Link")
	encodedURLWiki := util.URLJoin(markup.TestRepoURL, "wiki", "Link%3F")
	imgurlWiki := util.URLJoin(markup.TestRepoURL, "wiki", "raw", "Link.jpg")
	otherImgurlWiki := util.URLJoin(markup.TestRepoURL, "wiki", "raw", "Link+Other.jpg")
	encodedImgurlWiki := util.URLJoin(markup.TestRepoURL, "wiki", "raw", "Link+%23.jpg")
	notencodedImgurlWiki := util.URLJoin(markup.TestRepoURL, "wiki", "raw", "some", "path", "Link+#.jpg")
	favicon := "https://forgejo.org/favicon.ico"

	test(
		"[[Link]]",
		`<p><a href="`+url+`" rel="nofollow">Link</a></p>`,
		`<p><a href="`+urlWiki+`" rel="nofollow">Link</a></p>`)
	test(
		"[[Link.jpg]]",
		`<p><a href="`+imgurl+`" rel="nofollow"><img src="`+imgurl+`" title="Link.jpg" alt=""/></a></p>`,
		`<p><a href="`+imgurlWiki+`" rel="nofollow"><img src="`+imgurlWiki+`" title="Link.jpg" alt=""/></a></p>`)
	test(
		"[["+favicon+"]]",
		`<p><a href="`+favicon+`" rel="nofollow"><img src="`+favicon+`" title="favicon.ico" alt=""/></a></p>`,
		`<p><a href="`+favicon+`" rel="nofollow"><img src="`+favicon+`" title="favicon.ico" alt=""/></a></p>`)
	test(
		"[[Name|Link]]",
		`<p><a href="`+url+`" rel="nofollow">Name</a></p>`,
		`<p><a href="`+urlWiki+`" rel="nofollow">Name</a></p>`)
	test(
		"[[Name|Link.jpg]]",
		`<p><a href="`+imgurl+`" rel="nofollow"><img src="`+imgurl+`" title="Name" alt=""/></a></p>`,
		`<p><a href="`+imgurlWiki+`" rel="nofollow"><img src="`+imgurlWiki+`" title="Name" alt=""/></a></p>`)
	test(
		"[[Name|Link.jpg|alt=AltName]]",
		`<p><a href="`+imgurl+`" rel="nofollow"><img src="`+imgurl+`" title="AltName" alt="AltName"/></a></p>`,
		`<p><a href="`+imgurlWiki+`" rel="nofollow"><img src="`+imgurlWiki+`" title="AltName" alt="AltName"/></a></p>`)
	test(
		"[[Name|Link.jpg|title=Title]]",
		`<p><a href="`+imgurl+`" rel="nofollow"><img src="`+imgurl+`" title="Title" alt=""/></a></p>`,
		`<p><a href="`+imgurlWiki+`" rel="nofollow"><img src="`+imgurlWiki+`" title="Title" alt=""/></a></p>`)
	test(
		"[[Name|Link.jpg|alt=AltName|title=Title]]",
		`<p><a href="`+imgurl+`" rel="nofollow"><img src="`+imgurl+`" title="Title" alt="AltName"/></a></p>`,
		`<p><a href="`+imgurlWiki+`" rel="nofollow"><img src="`+imgurlWiki+`" title="Title" alt="AltName"/></a></p>`)
	test(
		"[[Name|Link.jpg|alt=\"AltName\"|title='Title']]",
		`<p><a href="`+imgurl+`" rel="nofollow"><img src="`+imgurl+`" title="Title" alt="AltName"/></a></p>`,
		`<p><a href="`+imgurlWiki+`" rel="nofollow"><img src="`+imgurlWiki+`" title="Title" alt="AltName"/></a></p>`)
	test(
		"[[Name|Link Other.jpg|alt=\"AltName\"|title='Title']]",
		`<p><a href="`+otherImgurl+`" rel="nofollow"><img src="`+otherImgurl+`" title="Title" alt="AltName"/></a></p>`,
		`<p><a href="`+otherImgurlWiki+`" rel="nofollow"><img src="`+otherImgurlWiki+`" title="Title" alt="AltName"/></a></p>`)
	test(
		"[[Link]] [[Other Link]]",
		`<p><a href="`+url+`" rel="nofollow">Link</a> <a href="`+otherURL+`" rel="nofollow">Other Link</a></p>`,
		`<p><a href="`+urlWiki+`" rel="nofollow">Link</a> <a href="`+otherURLWiki+`" rel="nofollow">Other Link</a></p>`)
	test(
		"[[Link?]]",
		`<p><a href="`+encodedURL+`" rel="nofollow">Link?</a></p>`,
		`<p><a href="`+encodedURLWiki+`" rel="nofollow">Link?</a></p>`)
	test(
		"[[Link]] [[Other Link]] [[Link?]]",
		`<p><a href="`+url+`" rel="nofollow">Link</a> <a href="`+otherURL+`" rel="nofollow">Other Link</a> <a href="`+encodedURL+`" rel="nofollow">Link?</a></p>`,
		`<p><a href="`+urlWiki+`" rel="nofollow">Link</a> <a href="`+otherURLWiki+`" rel="nofollow">Other Link</a> <a href="`+encodedURLWiki+`" rel="nofollow">Link?</a></p>`)
	test(
		"[[Link #.jpg]]",
		`<p><a href="`+encodedImgurl+`" rel="nofollow"><img src="`+encodedImgurl+`" title="Link #.jpg" alt=""/></a></p>`,
		`<p><a href="`+encodedImgurlWiki+`" rel="nofollow"><img src="`+encodedImgurlWiki+`" title="Link #.jpg" alt=""/></a></p>`)
	test(
		"[[Name|Link #.jpg|alt=\"AltName\"|title='Title']]",
		`<p><a href="`+encodedImgurl+`" rel="nofollow"><img src="`+encodedImgurl+`" title="Title" alt="AltName"/></a></p>`,
		`<p><a href="`+encodedImgurlWiki+`" rel="nofollow"><img src="`+encodedImgurlWiki+`" title="Title" alt="AltName"/></a></p>`)
	test(
		"[[some/path/Link #.jpg]]",
		`<p><a href="`+notencodedImgurl+`" rel="nofollow"><img src="`+notencodedImgurl+`" title="Link #.jpg" alt=""/></a></p>`,
		`<p><a href="`+notencodedImgurlWiki+`" rel="nofollow"><img src="`+notencodedImgurlWiki+`" title="Link #.jpg" alt=""/></a></p>`)
	test(
		"<p><a href=\"https://example.org\">[[foobar]]</a></p>",
		`<p><a href="https://example.org" rel="nofollow">[[foobar]]</a></p>`,
		`<p><a href="https://example.org" rel="nofollow">[[foobar]]</a></p>`)
}

func TestRender_RelativeImages(t *testing.T) {
	setting.AppURL = markup.TestAppURL

	test := func(input, expected, expectedWiki string) {
		buffer, err := markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base:       markup.TestRepoURL,
				BranchPath: "master",
			},
			Metas: localMetas,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(buffer)))
		buffer, err = markdown.RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: markup.TestRepoURL,
			},
			Metas:  localMetas,
			IsWiki: true,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expectedWiki), strings.TrimSpace(string(buffer)))
	}

	rawwiki := util.URLJoin(markup.TestRepoURL, "wiki", "raw")
	mediatree := util.URLJoin(markup.TestRepoURL, "media", "master")

	test(
		`<img src="Link">`,
		`<img src="`+util.URLJoin(mediatree, "Link")+`"/>`,
		`<img src="`+util.URLJoin(rawwiki, "Link")+`"/>`)

	test(
		`<img src="./icon.png">`,
		`<img src="`+util.URLJoin(mediatree, "icon.png")+`"/>`,
		`<img src="`+util.URLJoin(rawwiki, "icon.png")+`"/>`)
}

func Test_ParseClusterFuzz(t *testing.T) {
	setting.AppURL = markup.TestAppURL

	localMetas := map[string]string{
		"user": "go-gitea",
		"repo": "gitea",
	}

	data := "<A><maTH><tr><MN><bodY ÿ><temPlate></template><tH><tr></A><tH><d<bodY "

	var res strings.Builder
	err := markup.PostProcess(&markup.RenderContext{
		Ctx: git.DefaultContext,
		Links: markup.Links{
			Base: "https://example.com",
		},
		Metas: localMetas,
	}, strings.NewReader(data), &res)
	require.NoError(t, err)
	assert.NotContains(t, res.String(), "<html")

	data = "<!DOCTYPE html>\n<A><maTH><tr><MN><bodY ÿ><temPlate></template><tH><tr></A><tH><d<bodY "

	res.Reset()
	err = markup.PostProcess(&markup.RenderContext{
		Ctx: git.DefaultContext,
		Links: markup.Links{
			Base: "https://example.com",
		},
		Metas: localMetas,
	}, strings.NewReader(data), &res)

	require.NoError(t, err)
	assert.NotContains(t, res.String(), "<html")
}

func TestPostProcess_RenderDocument(t *testing.T) {
	setting.AppURL = markup.TestAppURL
	setting.StaticURLPrefix = markup.TestAppURL // can't run standalone

	localMetas := map[string]string{
		"user": "go-gitea",
		"repo": "gitea",
		"mode": "document",
	}

	test := func(input, expected string) {
		var res strings.Builder
		err := markup.PostProcess(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				AbsolutePrefix: true,
				Base:           "https://example.com",
			},
			Metas: localMetas,
		}, strings.NewReader(input), &res)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(res.String()))
	}

	// Issue index shouldn't be post processing in a document.
	test(
		"#1",
		"#1")

	// But cross-referenced issue index should work.
	test(
		"go-gitea/gitea#12345",
		`<a href="`+util.URLJoin(markup.TestAppURL, "go-gitea", "gitea", "issues", "12345")+`" class="ref-issue">go-gitea/gitea#12345</a>`)

	// Test that other post processing still works.
	test(
		":gitea:",
		`<span class="emoji" aria-label="gitea" data-alias="gitea"><img alt=":gitea:" src="`+setting.StaticURLPrefix+`/assets/img/emoji/gitea.png"/></span>`)
	test(
		"Some text with 😄 in the middle",
		`Some text with <span class="emoji" aria-label="grinning face with smiling eyes" data-alias="smile">😄</span> in the middle`)
	test("http://localhost:3000/person/repo/issues/4#issuecomment-1234",
		`<a href="http://localhost:3000/person/repo/issues/4#issuecomment-1234" class="ref-issue">person/repo#4 (comment)</a>`)
}

func TestIssue16020(t *testing.T) {
	setting.AppURL = markup.TestAppURL

	localMetas := map[string]string{
		"user": "go-gitea",
		"repo": "gitea",
	}

	data := `<img src="data:image/png;base64,i//V"/>`

	var res strings.Builder
	err := markup.PostProcess(&markup.RenderContext{
		Ctx:   git.DefaultContext,
		Metas: localMetas,
	}, strings.NewReader(data), &res)
	require.NoError(t, err)
	assert.Equal(t, data, res.String())
}

func BenchmarkEmojiPostprocess(b *testing.B) {
	data := "🥰 "
	for len(data) < 1<<16 {
		data += data
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res strings.Builder
		err := markup.PostProcess(&markup.RenderContext{
			Ctx:   git.DefaultContext,
			Metas: localMetas,
		}, strings.NewReader(data), &res)
		require.NoError(b, err)
	}
}

func TestFuzz(t *testing.T) {
	s := "t/l/issues/8#/../../a"
	renderContext := markup.RenderContext{
		Ctx: git.DefaultContext,
		Links: markup.Links{
			Base: "https://example.com/go-gitea/gitea",
		},
		Metas: map[string]string{
			"user": "go-gitea",
			"repo": "gitea",
		},
	}

	err := markup.PostProcess(&renderContext, strings.NewReader(s), io.Discard)

	require.NoError(t, err)
}

func TestIssue18471(t *testing.T) {
	data := `http://domain/org/repo/compare/783b039...da951ce`

	var res strings.Builder
	err := markup.PostProcess(&markup.RenderContext{
		Ctx:   git.DefaultContext,
		Metas: localMetas,
	}, strings.NewReader(data), &res)

	require.NoError(t, err)
	assert.Equal(t, "<a href=\"http://domain/org/repo/compare/783b039...da951ce\" class=\"compare\"><code class=\"nohighlight\">783b039...da951ce</code></a>", res.String())
}

func TestRender_FilePreview(t *testing.T) {
	defer test.MockVariableValue(&setting.StaticRootPath, "../../")()
	defer test.MockVariableValue(&setting.Names, []string{"english"})()
	defer test.MockVariableValue(&setting.Langs, []string{"en-US"})()
	translation.InitLocales(t.Context())

	setting.AppURL = markup.TestAppURL
	markup.Init(&markup.ProcessorHelper{
		GetRepoFileBlob: func(ctx context.Context, ownerName, repoName, commitSha, filePath string, language *string) (*git.Blob, error) {
			gitRepo, err := git.OpenRepository(git.DefaultContext, "./tests/repo/repo1_filepreview")
			require.NoError(t, err)
			defer gitRepo.Close()

			commit, err := gitRepo.GetCommit(commitSha)
			require.NoError(t, err)

			blob, err := commit.GetBlobByPath(filePath)
			require.NoError(t, err)

			return blob, nil
		},
	})

	sha := "190d9492934af498c3f669d6a2431dc5459e5b20"
	commitFilePreview := util.URLJoin(markup.TestRepoURL, "src", "commit", sha, "path", "to", "file.go") + "#L2-L3"

	testRender := func(input, expected string, metas map[string]string) {
		buffer, err := markup.RenderString(&markup.RenderContext{
			Ctx:          git.DefaultContext,
			RelativePath: ".md",
			Metas:        metas,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	t.Run("single", func(t *testing.T) {
		testRender(
			commitFilePreview,
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)
	})

	t.Run("cross-repo", func(t *testing.T) {
		testRender(
			commitFilePreview,
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/" rel="nofollow">gogits/gogs</a> – `+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">gogits/gogs@190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			map[string]string{
				"user": "gogits",
				"repo": "gogs2",
			},
		)
	})
	t.Run("single-line", func(t *testing.T) {
		testRender(
			util.URLJoin(markup.TestRepoURL, "src", "commit", "4c1aaf56bcb9f39dcf65f3f250726850aed13cd6", "single-line.txt")+"#L1",
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/" rel="nofollow">gogits/gogs</a> – `+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/4c1aaf56bcb9f39dcf65f3f250726850aed13cd6/single-line.txt#L1" class="muted" rel="nofollow">single-line.txt</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Line 1 in <a href="http://localhost:3000/gogits/gogs/src/commit/4c1aaf56bcb9f39dcf65f3f250726850aed13cd6" class="text black" rel="nofollow">gogits/gogs@4c1aaf5</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="1"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner">A`+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			map[string]string{
				"user": "gogits",
				"repo": "gogs2",
			},
		)
	})

	t.Run("AppSubURL", func(t *testing.T) {
		urlWithSub := util.URLJoin(markup.TestAppURL, "sub", markup.TestOrgRepo, "src", "commit", sha, "path", "to", "file.go") + "#L2-L3"

		testRender(
			urlWithSub,
			`<p><a href="http://localhost:3000/sub/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" rel="nofollow"><code>190d949293/path/to/file.go (L2-L3)</code></a></p>`,
			localMetas,
		)

		defer test.MockVariableValue(&setting.AppURL, markup.TestAppURL+"sub/")()
		defer test.MockVariableValue(&setting.AppSubURL, "/sub")()

		testRender(
			urlWithSub,
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/sub/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/sub/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)

		testRender(
			"first without sub "+commitFilePreview+" second "+urlWithSub,
			`<p>first without sub <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" rel="nofollow"><code>190d949293/path/to/file.go (L2-L3)</code></a> second </p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/sub/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/sub/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)
	})

	t.Run("multiples", func(t *testing.T) {
		testRender(
			"first "+commitFilePreview+" second "+commitFilePreview,
			`<p>first </p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p> second </p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)

		testRender(
			"first "+commitFilePreview+" second "+commitFilePreview+" third "+commitFilePreview,
			`<p>first </p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p> second </p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p> third </p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)
	})

	commitFileURL := util.URLJoin(markup.TestRepoURL, "src", "commit", "c9913120ed2c1e27c1d7752ecdb7a504dc7cf6be", "path", "to", "file.md")

	t.Run("rendered file with ?display=source", func(t *testing.T) {
		testRender(
			commitFileURL+"?display=source"+"#L1-L2",
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/c9913120ed2c1e27c1d7752ecdb7a504dc7cf6be/path/to/file.md?display=source#L1-L2" class="muted" rel="nofollow">path/to/file.md</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 1 to 2 in <a href="http://localhost:3000/gogits/gogs/src/commit/c9913120ed2c1e27c1d7752ecdb7a504dc7cf6be" class="text black" rel="nofollow">c991312</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="1"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="gh"># A`+"\n"+`</span></code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="gh"></span>B`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)
	})

	t.Run("rendered file without ?display=source", func(t *testing.T) {
		testRender(
			commitFileURL+"#L1-L2",
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/c9913120ed2c1e27c1d7752ecdb7a504dc7cf6be/path/to/file.md?display=source#L1-L2" class="muted" rel="nofollow">path/to/file.md</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 1 to 2 in <a href="http://localhost:3000/gogits/gogs/src/commit/c9913120ed2c1e27c1d7752ecdb7a504dc7cf6be" class="text black" rel="nofollow">c991312</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="1"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="gh"># A`+"\n"+`</span></code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="gh"></span>B`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)
	})

	commitFileURL = util.URLJoin(markup.TestRepoURL, "src", "commit", "190d9492934af498c3f669d6a2431dc5459e5b20", "path", "to", "file.go")

	t.Run("normal file with ?display=source", func(t *testing.T) {
		testRender(
			commitFileURL+"?display=source"+"#L2-L3",
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20/path/to/file.go?display=source#L2-L3" class="muted" rel="nofollow">path/to/file.go</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Lines 2 to 3 in <a href="http://localhost:3000/gogits/gogs/src/commit/190d9492934af498c3f669d6a2431dc5459e5b20" class="text black" rel="nofollow">190d949</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="2"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">B</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="3"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner"><span class="nx">C</span>`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)
	})

	commitFileURL = util.URLJoin(markup.TestRepoURL, "src", "commit", "eeb243c3395e1921c5d90e73bd739827251fc99d", "path", "to", "file%20%23.txt")

	t.Run("file with strange characters in name", func(t *testing.T) {
		testRender(
			commitFileURL+"#L1",
			`<p></p>`+
				`<div class="file-preview-box">`+
				`<div class="header">`+
				`<div>`+
				`<a href="http://localhost:3000/gogits/gogs/src/commit/eeb243c3395e1921c5d90e73bd739827251fc99d/path/to/file%20%23.txt#L1" class="muted" rel="nofollow">path/to/file #.txt</a>`+
				`</div>`+
				`<span class="text small grey">`+
				`Line 1 in <a href="http://localhost:3000/gogits/gogs/src/commit/eeb243c3395e1921c5d90e73bd739827251fc99d" class="text black" rel="nofollow">eeb243c</a>`+
				`</span>`+
				`</div>`+
				`<div class="ui table">`+
				`<table class="file-preview">`+
				`<tbody>`+
				`<tr>`+
				`<td class="lines-num"><span data-line-number="1"></span></td>`+
				`<td class="lines-code chroma"><code class="code-inner">A`+"\n"+`</code></td>`+
				`</tr>`+
				`</tbody>`+
				`</table>`+
				`</div>`+
				`</div>`+
				`<p></p>`,
			localMetas,
		)
	})
}
