// package meta is a extension for the goldmark(http://github.com/yuin/goldmark).
//
// This extension parses YAML metadata blocks and store metadata to a
// parser.Context.
package meta

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"notabug.org/gearsix/dati"
)

type metadata map[string]interface{}

type data struct {
	Map   metadata
	Error error
	Node  gast.Node
}

var contextKey = parser.NewContextKey()

// Get returns a metadata.
func Get(pc parser.Context) metadata {
	v := pc.Get(contextKey)
	if v == nil {
		return nil
	}
	d := v.(*data)
	return d.Map
}

// TryGet tries to get a metadata.
// If there are parsing errors, then nil and error are returned
func TryGet(pc parser.Context) (metadata, error) {
	dtmp := pc.Get(contextKey)
	if dtmp == nil {
		return nil, nil
	}
	d := dtmp.(*data)
	if d.Error != nil {
		return nil, d.Error
	}
	return d.Map, nil
}

const openToken = "<!--"
const closeToken = "-->"
const formatYaml = ':'
const formatToml = '#'
const formatJsonOpen = '{'
const formatJsonClose = '}'

type metaParser struct {
	format byte
}

var defaultParser = &metaParser{}

// NewParser returns a BlockParser that can parse metadata blocks.
func NewParser() parser.BlockParser {
	return defaultParser
}

func isOpen(line []byte) bool {
	line = util.TrimRightSpace(util.TrimLeftSpace(line))
	for i := 0; i < len(line); i++ {
		if len(line[i:]) >= len(openToken)+1 && line[i] == openToken[0] {
			signal := line[i+len(openToken)]
			switch signal {
			case formatYaml:
				fallthrough
			case formatToml:
				fallthrough
			case formatJsonOpen:
				return true
			default:
				break
			}
		}
	}
	return false
}

// isClose will check `line` for the closing token.
// If found, the integer returned will be the *nth* byte of `line` that the close token starts at.
// If not found, then -1 is returned.
func isClose(line []byte, signal byte) int {
	//line = util.TrimRightSpace(util.TrimLeftSpace(line))
	for i := 0; i < len(line); i++ {
		if line[i] == signal && len(line[i:]) >= len(closeToken)+1 {
			i++
			if string(line[i:i+len(closeToken)]) == closeToken {
				if signal == formatJsonClose {
					return i
				} else {
					return i - 1
				}
			}
		}
	}
	return -1
}

func (b *metaParser) Trigger() []byte {
	return []byte{openToken[0]}
}

func (b *metaParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	if linenum, _ := reader.Position(); linenum != 0 {
		return nil, parser.NoChildren
	}
	line, _ := reader.PeekLine()

	if isOpen(line) {
		reader.Advance(len(openToken))
		if b.format = reader.Peek(); b.format == formatJsonOpen {
			b.format = formatJsonClose
		} else {
			reader.Advance(1)
		}

		node := gast.NewTextBlock()
		if b.Continue(node, reader, pc) != parser.Close {
			return node, parser.NoChildren
		}
		parent.AppendChild(parent, node)
		b.Close(node, reader, pc)
	}
	return nil, parser.NoChildren
}

func (b *metaParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if n := isClose(line, b.format); n != -1 && !util.IsBlank(line) {
		segment.Stop -= len(line[n:])
		node.Lines().Append(segment)
		reader.Advance(n + len(closeToken) + 1)
		return parser.Close
	}
	node.Lines().Append(segment)
	return parser.Continue | parser.NoChildren
}

func (b *metaParser) loadMetadata(buf []byte) (meta metadata, err error) {
	var format dati.DataFormat
	switch b.format {
	case formatYaml:
		format = dati.YAML
	case formatToml:
		format = dati.TOML
	case formatJsonClose:
		format = dati.JSON
	default:
		return meta, dati.ErrUnsupportedData(string(b.format))
	}
	err = dati.LoadData(format, bytes.NewReader(buf), &meta)
	return meta, err
}

func (b *metaParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	lines := node.Lines()
	var buf bytes.Buffer
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		buf.Write(segment.Value(reader.Source()))
	}
	d := &data{Node: node}
	d.Map, d.Error = b.loadMetadata(buf.Bytes())

	pc.Set(contextKey, d)

	if d.Error == nil {
		node.Parent().RemoveChild(node.Parent(), node)
	}
}

func (b *metaParser) CanInterruptParagraph() bool {
	return true
}

func (b *metaParser) CanAcceptIndentedLine() bool {
	return true
}

type astTransformer struct {
	transformerConfig
}

type transformerConfig struct {
	// Stores metadata in ast.Document.Meta().
	StoresInDocument bool
}

type transformerOption interface {
	Option

	// SetMetaOption sets options for the metadata parser.
	SetMetaOption(*transformerConfig)
}

var _ transformerOption = &withStoresInDocument{}

type withStoresInDocument struct {
	value bool
}

// WithStoresInDocument is a functional option that parser will store meta in ast.Document.Meta().
func WithStoresInDocument() Option {
	return &withStoresInDocument{
		value: true,
	}
}

func newTransformer(opts ...transformerOption) parser.ASTTransformer {
	p := &astTransformer{
		transformerConfig: transformerConfig{
			StoresInDocument: false,
		},
	}
	for _, o := range opts {
		o.SetMetaOption(&p.transformerConfig)
	}
	return p
}

func (a *astTransformer) Transform(node *gast.Document, reader text.Reader, pc parser.Context) {
	dtmp := pc.Get(contextKey)
	if dtmp == nil {
		return
	}
	d := dtmp.(*data)
	if d.Error != nil {
		msg := gast.NewString([]byte(fmt.Sprintf("<!-- meta error, %s -->", d.Error)))
		msg.SetCode(true)
		d.Node.AppendChild(d.Node, msg)
		return
	}

	if a.StoresInDocument {
		for k, v := range d.Map {
			node.AddMeta(k, v)
		}
	}
}

// Option interface sets options for this extension.
type Option interface {
	metaOption()
}

func (o *withStoresInDocument) metaOption() {}

func (o *withStoresInDocument) SetMetaOption(c *transformerConfig) {
	c.StoresInDocument = o.value
}

type meta struct {
	options []Option
}

// Meta is a extension for the goldmark.
var Meta = &meta{}

// New returns a new Meta extension.
func New(opts ...Option) goldmark.Extender {
	e := &meta{
		options: opts,
	}
	return e
}

// Extend implements goldmark.Extender.
func (e *meta) Extend(m goldmark.Markdown) {
	topts := []transformerOption{}
	for _, opt := range e.options {
		if topt, ok := opt.(transformerOption); ok {
			topts = append(topts, topt)
		}
	}
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewParser(), 0),
		),
	)
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(newTransformer(topts...), 0),
		),
	)
}
