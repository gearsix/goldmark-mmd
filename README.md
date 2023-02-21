---
Problem: In some parsers this will be rendered.
Solution: mmd
---

mmd
===

*Fork of the [goldmark-meta](http://github.com/yuin/goldmark-meta) extension for [goldmark](http://github.com/yuin/goldmark), both by [yuin](http://github.com/yuin).*

This extension provides parsing of metadata in markdown using a different syntax than most to avoid the metadata being improperly parsed by any markdown renderer that doesn't parse for metadata.

Motivation
----------

Most extensions for markdown that provide parsing for document metadata use YAML with its [Document Markers](https://yaml.org/spec/1.2.2/#912-document-markers) syntax.
This is the idiomatic way of parsing metadata for in a markdown document, since [jekyll](https://jekyllrb.com/docs/front-matter/) started using it.

[Hugo](gohugo.io) (a tool similair to jekyll) later decided it would allow for multiple data formats, which required different document markers for those languages.
For JSON the marker was simply the opening `{` and for TOML it was `+++`.

The are great solutions but all of them also render in any markdown parser (e.g. GitHub) that doesn't expect metadata, causing an awkward section at the top of the render (see the top of this document).

To deal with this, the original *goldmark-meta* extension added a bunch of code that gave the option to render metdata as a HTML table but my argument is that metadata shouldn't be rendered at all.

This extension uses a different syntax to avoid the issue by putting the metadata in a HTML comment, that way it doesn't get rendered regardless of the markdown parser - which is how metadata should be treated.


Syntax
------

Metadata must start of the buffer using an **opening HTML comment tag** (`<!--`), followed by a **signal character** (signalling the metadata format being used).

Metadata must end with the same **signal character** followed by  **closing HTML comment tag** (`-->`).

Everything inbetween these two tags should be the metadata in the signalled syntax.

### Signal Characters

- YAML = `:`&emsp;(`<!--:...:-->`)
- TOML = `#`&emsp;(`<!--#...#-->`)
- JSON = `{}`&emsp;(`<!--{...}-->`)


Usage
-----

This is an extension to [goldmark](http://github.com/yuin/goldmark), so it'll need to be used in conjuction with it.

### Installation

```
go get github.com/gearsix/mmd
```


### Access the metadata

```go
import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/gearsix/mmd"
)

func main() {
	markdown := goldmark.New(goldmark.WithExtensions(mmd.MarkdownMeta))
	source := `<!--:
Title: Markdown Meta
Tags:
	- markdown
	- goldmark
:-->

This is an example of markdown meta using YAML.
`

	var buf bytes.Buffer
	context := parser.NewContext()
	if err := markdown.Convert([]byte(source), &buf, parser.WithContext(context)); err != nil {
		panic(err)
	}
	metaData := mmd.Get(context)
	title := metaData["Title"]
	fmt.Print(title)
}
```

Or `WithStoresInDocument` option:

```go
import (
	"bytes"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/gearsix/mmd"
)

func main() {
	markdown := goldmark.New(goldmark.WithExtensions(
		meta.New(mmd.WithStoresInDocument()),
	))
	source := `<!--{ "Title": "Markdown Meta", "Tags": [ "markdown", "goldmark" ] }-->
This is an example of markdown meta using JSON.
`

	document := markdown.Parser().Parse(text.NewReader([]byte(source)))
	metaData := document.OwnerDocument().MarkdownMeta()
	title := metaData["Title"]
	fmt.Print(title)
}
```

License
-------
MIT

Authors
-------

- Yusuke Inuzuka
- gearsix
