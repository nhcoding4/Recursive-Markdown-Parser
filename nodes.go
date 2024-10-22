package main

import (
	"bytes"
	"fmt"
)

// ----------------------------------------------------------------------------------------------------------------
// Types of Nodes used in the AST.
// ----------------------------------------------------------------------------------------------------------------

type NodeType string

const (
	Code          NodeType = "code"
	Image         NodeType = "img"
	Link          NodeType = "a"
	Bold          NodeType = "b"
	Div           NodeType = "div"
	Heading1      NodeType = "h1"
	Heading2      NodeType = "h2"
	Heading3      NodeType = "h3"
	Heading4      NodeType = "h4"
	Heading5      NodeType = "h5"
	Heading6      NodeType = "h6"
	Italic        NodeType = "i"
	ListElement   NodeType = "li"
	OrderedList   NodeType = "ol"
	Paragraph     NodeType = "p"
	Quote         NodeType = "blockquote"
	UnorderedList NodeType = "ul"
	PlainText     NodeType = ""
	Escaped       NodeType = "escaped"
)

// ----------------------------------------------------------------------------------------------------------------

type HtmlNode interface {
	toHtml() string
}

// ----------------------------------------------------------------------------------------------------------------
// LeafNode
// ----------------------------------------------------------------------------------------------------------------

type LeafType string

type LeafNode struct {
	value      string
	nodeType   NodeType
	properties map[string]string
}

func NewLeafNode(value *string, nodeType *NodeType, properties *map[string]string) *LeafNode {
	newLeaf := &LeafNode{
		value:    *value,
		nodeType: *nodeType,
	}

	if properties != nil {
		newLeaf.properties = *properties
	}

	return newLeaf
}

func (l *LeafNode) propertiesToHtml() string {
	var out bytes.Buffer

	for key, value := range l.properties {
		out.WriteString(fmt.Sprintf(" %v=%v", key, value))
	}

	return out.String()
}

func (l *LeafNode) toHtml() string {
	switch l.nodeType {
	case Code, Link:
		return fmt.Sprintf("<%v%v>%v</%v>", l.nodeType, l.propertiesToHtml(), l.value, l.nodeType)
	case Image:
		return fmt.Sprintf("<%v%v/>", l.nodeType, l.propertiesToHtml())
	default:
		return l.value
	}
}

// ----------------------------------------------------------------------------------------------------------------
// ParentNode
// ----------------------------------------------------------------------------------------------------------------

type ParentNode struct {
	nodeType   NodeType
	childNodes []HtmlNode
}

func NewParentNode(nodeType *NodeType, childNodes ...HtmlNode) *ParentNode {
	return &ParentNode{
		nodeType:   *nodeType,
		childNodes: childNodes,
	}
}

func (p *ParentNode) toHtml() string {
	var out bytes.Buffer

	if p.nodeType != PlainText {
		out.WriteString(fmt.Sprintf("<%v>", p.nodeType))
	}

	for _, child := range p.childNodes {
		out.WriteString(child.toHtml())
	}

	if p.nodeType != PlainText {
		out.WriteString(fmt.Sprintf("</%v>", p.nodeType))
	}
	return out.String()
}

// ----------------------------------------------------------------------------------------------------------------
