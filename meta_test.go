package meta

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
)

var testMetaFormats = []string{"yaml", "json", "toml"}
var validSource = map[string]string{
	"yaml": `<!--:
Title: mmd
Summary: Add YAML metadata to the document
Tags:
  - markdown
  - goldmark
:-->

Markdown with metadata
`,
	"json": `<!--{ "Title": "mmd", "Summary": "Add JSON metadata to the document", "Tags": [ "markdown", "goldmark" ] }-->
Markdown with metadata`,
	"toml": `<!--# Title = "mmd"
		Summary = "Add TOML metadata to the document"
		Tags = [ "markdown", "goldmark" ] #-->
Markdown with metadata
`,
}
var invalidSource = map[string]string{
	"yaml": `<!--:
Title: mmd
Summary: Add YAML metadata to the document
Tags:
- : {
}
  - markdown
  - goldmark
:-->

Markdown with metadata`,
	"json": `<!--{ "Title:" "mmd", "Summary": "Add JSON metadata to the document", "Tags": [ "markdown", "goldmark" ] }-->
Markdown with metadata`,
	"toml": `<!--# Title = "mmd"
		Summary = "Add TOML metadata to the document
		Tags == [ markdown", "goldmark ] #-->
Markdown with metadata
`,
}

func TestMeta(t *testing.T) {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			Meta,
		),
	)

	context := parser.NewContext()

	for _, format := range testMetaFormats {
		var buf bytes.Buffer
		if err := markdown.Convert([]byte(validSource[format]), &buf, parser.WithContext(context)); err != nil {
			t.Fatal(err)
		}

		metaData := Get(context)

		title := metaData["Title"]
		if s, ok := title.(string); !ok {
			t.Errorf("%s: Title not found in meta data or is not a string", format)
		} else if s != "mmd" {
			t.Errorf("%s: Title must be 'mmd', but got %v", format, s)
		}

		if buf.String() != "<p>Markdown with metadata</p>\n" {
			t.Errorf("%s: should render '<p>Markdown with metadata</p>', but '%s'", format, buf.String())
		}

		if tags, ok := metaData["Tags"].([]interface{}); !ok {
			t.Errorf("%s: Tags not found in meta data or is not a slice", format)
		} else if len(tags) != 2 {
			t.Errorf("%s: Tags must be a slice that has 2 elements", format)
		} else if tags[0] != "markdown" {
			t.Errorf("%s: Tag#1 must be 'markdown', but got %s", format, tags[0])
		} else if tags[1] != "goldmark" {
			t.Errorf("%s: Tag#2 must be 'goldmark', but got %s", format, tags[1])
		}
	}
}

func TestMetaError(t *testing.T) {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			New(),
		),
	)

	var buf bytes.Buffer
	context := parser.NewContext()

	if err := markdown.Convert([]byte(invalidSource["json"]), &buf, parser.WithContext(context)); err != nil {
		t.Fatal(err)
	}
	if buf.String() != `<!-- json: line 1: TODO -->
<p>This is a markdown with JSON metadata</p>
` {
		t.Error("json: invalid error output")
	}

	if err := markdown.Convert([]byte(invalidSource["yaml"]), &buf, parser.WithContext(context)); err != nil {
		t.Fatal(err)
	}
	if buf.String() != `<!-- yaml: line 3: did not find expected key -->
<p>This is markdown with YAML metadata</p>
` {
		t.Error("yaml: invalid error output")
	}

	if err := markdown.Convert([]byte(invalidSource["toml"]), &buf, parser.WithContext(context)); err != nil {
		t.Fatal(err)
	}
	if buf.String() != `<!-- toml: line 3: did not find expected key -->
<p>Markdown with metadata</p>
` {
		t.Error("toml: invalid error output")
	}

	if v, err := TryGet(context); err == nil {
		t.Error("error should not be nil")
	} else if v != nil {
		t.Error("data should be nil when there are errors")
	}
}
