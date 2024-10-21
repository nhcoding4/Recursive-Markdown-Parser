package main

import "fmt"

// ----------------------------------------------------------------------------------------------------------------

const EOF rune = 0

// ----------------------------------------------------------------------------------------------------------------
// Parser Types.
// ----------------------------------------------------------------------------------------------------------------

type ParseFunc func() HtmlNode

type Parser struct {
	input        string
	current      int
	peek         int
	ch           rune
	parsingFuncs map[NodeType]ParseFunc
}

// ----------------------------------------------------------------------------------------------------------------

func NewParser(input *string) *Parser {
	newParser := &Parser{
		input:        *input,
		current:      0,
		peek:         0,
		parsingFuncs: make(map[NodeType]ParseFunc),
	}
	newParser.readChar()

	newParser.registerFunc(Bold, newParser.boldItalicNode)
	newParser.registerFunc(Italic, newParser.boldItalicNode)
	newParser.registerFunc(Image, func() HtmlNode { return newParser.parseImageLink(Image) })
	newParser.registerFunc(Link, func() HtmlNode { return newParser.parseImageLink(Link) })
	newParser.registerFunc(Code, newParser.parseCode)
	newParser.registerFunc(UnorderedList, newParser.parseUnorderedList)
	newParser.registerFunc(OrderedList, newParser.parseOrderedList)
	newParser.registerFunc(Escaped, newParser.parseEscaped)

	return newParser
}

// ----------------------------------------------------------------------------------------------------------------
// Parsing functions
// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parse() HtmlNode {
	rootType := p.blockType()
	p.consumeBlockHeading(rootType)

	start := p.current
	children := make([]HtmlNode, 0)

	for p.ch != EOF {
		p.buildNestedorRead(&start, &children)
	}

	if start < p.current {
		children = append(children, NewLeafNode(p.input[start:p.peek], PlainText, nil))
	}

	return NewParentNode(rootType, children...)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseChildren(identSize int, breakCondition func() bool, nodeType NodeType) HtmlNode {
	functionStart := p.current
	p.readX(identSize)
	start := p.current
	children := make([]HtmlNode, 0)

	for p.ch != EOF {
		if breakCondition() {
			break
		}
		p.buildNestedorRead(&start, &children)
	}

	if p.checkUnterminatedExceptions(nodeType) {
		return NewParentNode(PlainText,
			NewLeafNode(p.input[functionStart:p.peek], PlainText, nil))
	}

	offset := 0
	if nodeType == Paragraph {
		offset++
	}
	if start < p.current {
		children = append(children, NewLeafNode(p.input[start:p.current+offset], PlainText, nil))
	}

	p.readX(identSize)

	return NewParentNode(nodeType, children...)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseBold() HtmlNode {
	identSize := 2
	breakCondition := func() bool {
		return p.ch == '*' && p.peekCharX(0) == '*'
	}

	return p.parseChildren(identSize, breakCondition, Bold)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseCode() HtmlNode {
	identSize := 3
	breakCondition := func() bool {
		return p.ch == '`' && p.peekCharX(0) == '`' && p.peekCharX(1) == '`'
	}

	return p.parseChildren(identSize, breakCondition, Code)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseEscaped() HtmlNode {
	p.readChar()
	newLeaf := NewLeafNode(string(p.input[p.current]), PlainText, nil)
	p.readChar()
	return newLeaf
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseItalic() HtmlNode {
	identSize := 1
	breakCondition := func() bool {
		return p.ch == '*' && p.peekCharX(0) != '*'
	}

	return p.parseChildren(identSize, breakCondition, Italic)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseImageLink(nodeType NodeType) HtmlNode {
	functionStart := p.current

	if nodeType == Image {
		p.readChar()
	}

	breakConditionPrefix := func() bool {
		return p.ch == ']'
	}
	prefixNode := p.parseChildren(1, breakConditionPrefix, PlainText)
	if p.ch != '(' {
		return NewLeafNode(p.input[functionStart:p.current], PlainText, nil)
	}

	breakCondtionSrc := func() bool {
		return p.ch == ')'
	}
	srcNode := p.parseChildren(1, breakCondtionSrc, PlainText)

	textNode := p.imageLinkChecks(functionStart)
	if textNode != nil {
		return textNode
	}

	switch nodeType {
	case Image:
		return NewLeafNode("", Image, map[string]string{
			"alt": prefixNode.toHtml(),
			"src": srcNode.toHtml(),
		})
	case Link:
		return NewLeafNode(prefixNode.toHtml(), Link, map[string]string{
			"href": srcNode.toHtml(),
		})
	default:
		return nil
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseOrderedList() HtmlNode {
	consumeIdent := func() {
		for p.isDigit(p.ch) {
			p.readChar()
		}
		remainingIdent := 2
		p.readX(remainingIdent)
	}
	consumeIdent()

	breakCondition := func() bool {
		return p.isOrderedIdent()
	}

	return p.parseChildren(0, breakCondition, ListElement)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseUnorderedList() HtmlNode {
	identSize := 2
	p.readX(identSize)

	breakCondition := func() bool {
		return p.ch == '-' && p.peekCharX(0) == ' '
	}

	return p.parseChildren(0, breakCondition, ListElement)
}

// ----------------------------------------------------------------------------------------------------------------
// Helper functions.
// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) blockType() NodeType {
	switch p.ch {
	case '>':
		if p.peekCharX(0) == ' ' {
			return Quote
		}
		return Paragraph

	case '-':
		if p.peekCharX(0) == ' ' {
			return UnorderedList
		}
		return Paragraph

	case '#':
		return p.headingType()

	default:
		if p.isDigit(p.ch) {
			if p.isOrderedIdent() {
				return OrderedList
			}
			return Paragraph
		}
		return Paragraph
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) boldItalicNode() HtmlNode {
	if p.checkBoldNode() {
		return p.parseBold()
	}
	return p.parseItalic()
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) buildNestedorRead(start *int, targetSlice *[]HtmlNode) {
	ident := p.isIdent()
	if ident == Escaped {
		if *start < p.current {
			*targetSlice = append(*targetSlice, NewLeafNode(p.input[*start:p.current], PlainText, nil))
		}
		p.readChar()
		*targetSlice = append(*targetSlice, NewLeafNode(string(p.input[p.current]), PlainText, nil))
		p.readChar()
		*start = p.current

	} else if ident != PlainText {
		parseFunc := p.parsingFuncs[ident]

		if *start < p.current {
			*targetSlice = append(*targetSlice, NewLeafNode(p.input[*start:p.current], PlainText, nil))
		}
		*targetSlice = append(*targetSlice, parseFunc())
		*start = p.current

	} else {
		p.readChar()
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) checkBoldNode() bool {
	return p.ch == '*' && p.peekCharX(0) == '*' && p.peekCharX(1) != '*'
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) checkUnterminatedExceptions(nodeType NodeType) bool {
	case1 := p.ch == EOF
	case2 := nodeType != Quote
	case3 := nodeType != ListElement
	case4 := func() bool {
		switch nodeType {
		case Heading1, Heading2, Heading3, Heading4, Heading5, Heading6:
			return false
		default:
			return true
		}
	}()

	return case1 && case2 && case3 && case4
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) consumeBlockHeading(blockType NodeType) {
	switch blockType {
	case Quote:
		p.readX(2)
	case Heading1:
		p.readX(2)
	case Heading2:
		p.readX(3)
	case Heading3:
		p.readX(4)
	case Heading4:
		p.readX(5)
	case Heading5:
		p.readX(6)
	case Heading6:
		p.readX(7)
	default:
		return
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) headingType() NodeType {
	index := 0

	for {
		current := p.readFromIndex(index)
		if current != '#' || current == EOF {
			break
		}
		index++
	}

	if index >= 6 {
		return Heading6
	}
	return NodeType(fmt.Sprintf("h%v", index))
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) imageLinkChecks(functionStart int) HtmlNode {
	if p.ch == EOF {
		if p.peekPreviousX(0) != ')' {
			return NewLeafNode(p.input[functionStart:p.peek], PlainText, nil)
		}
	} else {
		if p.peekPreviousX(1) != ')' {
			return NewLeafNode(p.input[functionStart:p.peek], PlainText, nil)
		}
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) isIdent() NodeType {
	switch p.ch {
	case '*':
		if p.peekCharX(0) == '*' {
			return Bold
		}
		return Italic
	case '`':
		if p.peekCharX(0) == '`' && p.peekCharX(1) == '`' {
			return Code
		}
		return PlainText
	case '!':
		if p.peekCharX(0) == '[' {
			return Image
		}
		return PlainText
	case '[':
		return Link
	case '-':
		if p.peekCharX(0) == ' ' {
			return UnorderedList
		}
		return PlainText
	case '\\':
		return Escaped
	default:
		if p.isDigit(p.ch) {
			if p.isOrderedIdent() {
				return OrderedList
			}
			return PlainText
		}
		return PlainText
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) isOrderedIdent() bool {
	index := p.current

	for p.isDigit(p.readFromIndex(index)) {
		index++
	}

	if p.readFromIndex(index) == '.' && p.readFromIndex(index+1) == ' ' {
		return true
	}
	return false
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) peekCharX(amount int) rune {
	offset := p.peek + amount
	if offset >= len(p.input) {
		return EOF
	}
	return rune(p.input[offset])
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) peekPreviousX(amount int) rune {
	offset := p.current - amount
	if offset < 0 {
		return rune(p.input[0])
	}
	if offset >= len(p.input) {
		return EOF
	}
	return rune(p.input[offset])
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) readChar() {
	if p.peek >= len(p.input) {
		p.ch = EOF
		return
	}
	p.current = p.peek
	p.ch = rune(p.input[p.current])
	p.peek++
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) readFromIndex(index int) rune {
	if index >= len(p.input) {
		return EOF
	}
	return rune(p.input[index])
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) readX(total int) {
	for range total {
		p.readChar()
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) registerFunc(nodeType NodeType, fn ParseFunc) {
	p.parsingFuncs[nodeType] = fn
}

// ----------------------------------------------------------------------------------------------------------------
