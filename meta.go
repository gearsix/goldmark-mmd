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

	"gopkg.in/yaml.v2"
)

type data struct {
	Map   map[string]interface{}
	Items yaml.MapSlice
	Error error
	Node  gast.Node
}

var contextKey = parser.NewContextKey()

// Option interface sets options for this extension.
type Option interface {
	metaOption()
}

// Get returns a metadata.
func Get(pc parser.Context) map[string]interface{} {
	v := pc.Get(contextKey)
	if v == nil {
		return nil
	}
	d := v.(*data)
	return d.Map
}

// TryGet tries to get a metadata.
// If there are parsing errors, then nil and error are returned
func TryGet(pc parser.Context) (map[string]interface{}, error) {
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
const formatJson = '{'

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
			case formatJson:
				return true
			default:
				break
			}
		}
	}
	return false
}

func isClose(line []byte, signal byte) bool {
	line = util.TrimRightSpace(util.TrimLeftSpace(line))
	for i := 0; i < len(line); i++ {
		if len(line[:i]) > len(closeToken)+1 && line[i] == signal {
			i++
			if string(line[i:i+len(closeToken)]) == closeToken {
				return true
			}
		}
	}
	return false
}

func (b *metaParser) Trigger() []byte {
	return []byte{openToken[0]}
}

func (b *metaParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	linenum, _ := reader.Position()
	if linenum != 0 {
		return nil, parser.NoChildren
	}
	line, _ := reader.PeekLine()
	if isOpen(line) {
		reader.Advance(len(openToken))
		b.format = reader.Peek()
		reader.Advance(1)
		return gast.NewTextBlock(), parser.NoChildren
	}
	return nil, parser.NoChildren
}

func (b *metaParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if isClose(line, b.format) && !util.IsBlank(line) {
		reader.Advance(segment.Len())
		return parser.Close
	}
	node.Lines().Append(segment)
	return parser.Continue | parser.NoChildren
}

// TODO: bookmark
func (b *metaParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	lines := node.Lines()
	var buf bytes.Buffer
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		buf.Write(segment.Value(reader.Source()))
	}
	d := &data{}
	d.Node = node
	meta := map[string]interface{}{}
	if err := yaml.Unmarshal(buf.Bytes(), &meta); err != nil {
		d.Error = err
	} else {
		d.Map = meta
	}

	metaMapSlice := yaml.MapSlice{}
	if err := yaml.Unmarshal(buf.Bytes(), &metaMapSlice); err != nil {
		d.Error = err
	} else {
		d.Items = metaMapSlice
	}

	pc.Set(contextKey, d)

	if d.Error == nil {
		node.Parent().RemoveChild(node.Parent(), node)
	}
}

func (b *metaParser) CanInterruptParagraph() bool {
	return false
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

func (o *withStoresInDocument) metaOption() {}

func (o *withStoresInDocument) SetMetaOption(c *transformerConfig) {
	c.StoresInDocument = o.value
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
		msg := gast.NewString([]byte(fmt.Sprintf("<!-- %s -->", d.Error)))
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
