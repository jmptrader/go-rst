// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.

package parse

// NodeType identifies the type of a parse tree node.
type NodeType int

const (
	NodeSection NodeType = iota
	NodeParagraph
	NodeBlankLine
	NodeAdornment
)

var nodeTypes = [...]string{
	"NodeSection",
	"NodeParagraph",
	"NodeBlankLine",
	"NodeAdornment",
}

func (n NodeType) Type() NodeType {
	return n
}

func (n NodeType) String() string {
	return nodeTypes[n]
}

func (n NodeType) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

type Node interface {
	LineNumber() Line
	NodeType() NodeType
	Position() StartPosition
}

type NodeList []Node

func newList() *NodeList {
	return new(NodeList)
}

func (l *NodeList) append(n Node) {
	*l = append(*l, n)

}

type SectionNode struct {
	Type          NodeType `json:"node-type"`
	Text          string   `json:"text"`
	Level         int      `json:"level"`
	Length        int      `json:"length"`
	StartPosition `json:"start-position"`
	Line          `json:"line"`
	OverLine      *AdornmentNode `json:"overline"`
	UnderLine     *AdornmentNode `json:"underline"`
	Nodes         NodeList       `json:"node-list"`
}

func (s *SectionNode) NodeType() NodeType {
	return s.Type
}

func newSection(item item, level int, overAdorn item, underAdorn item) *SectionNode {
	n := &SectionNode{Text: item.Value.(string),
		Type:          NodeSection,
		Level:         level,
		StartPosition: item.StartPosition,
		Length:        item.Length,
	}

	if overAdorn.Value != nil {
		oRune := rune(overAdorn.Value.(string)[0])
		n.OverLine = &AdornmentNode{
			Char:          oRune,
			Type:          NodeAdornment,
			StartPosition: overAdorn.StartPosition,
			Line:          overAdorn.Line,
			Length:        overAdorn.Length,
		}
	}

	uRune := rune(underAdorn.Value.(string)[0])
	n.UnderLine = &AdornmentNode{
		Char:          uRune,
		Type:          NodeAdornment,
		StartPosition: underAdorn.StartPosition,
		Line:          underAdorn.Line,
		Length:        underAdorn.Length,
	}

	return n
}

type AdornmentNode struct {
	Char          rune     `json:"char"`
	Length        int      `json:"length"`
	Type          NodeType `json:"node-type"`
	Line          `json:"line"`
	StartPosition `json:"position"`
}

func (a AdornmentNode) NodeType() NodeType {
	return a.Type
}

func newBlankLine(i item) *BlankLineNode {
	return &BlankLineNode{
		Type:          NodeBlankLine,
		StartPosition: i.StartPosition,
		Line:          i.Line,
	}
}

type BlankLineNode struct {
	Type          NodeType `json:"node-type"`
	Line          `json:"line"`
	StartPosition `json:"position"`
}

func (b BlankLineNode) NodeType() NodeType {
	return b.Type
}

type ParagraphNode struct {
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Type          NodeType `json:"node-type"`
	Line          `json:"line"`
	StartPosition `json:"position"`
}

func newParagraph(i item) *ParagraphNode {
	return &ParagraphNode{
		Text:          i.Value.(string),
		Type:          NodeParagraph,
		StartPosition: i.StartPosition,
		Line:          i.Line,
		Length:        i.Length,
	}
}

func (p ParagraphNode) NodeType() NodeType {
	return p.Type
}
