package main

import "fmt"

// ----------------------------------------------------------------------------------------------------------------

const EOF rune = 0
const PeekOnce = 0
const PeekTwice = 1

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
	newParser.registerFunc(Image, func() HtmlNode { nodeType := Image; return newParser.parseImageLink(&nodeType) })
	newParser.registerFunc(Link, func() HtmlNode { nodeType := Link; return newParser.parseImageLink(&nodeType) })
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
		value := p.input[start:p.peek]
		nodeType := PlainText
		children = append(children, NewLeafNode(&value, &nodeType, nil))
	}

	return NewParentNode(&rootType, children...)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseChildren(identSize *int, breakCondition func() bool, nodeType *NodeType) HtmlNode {
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
		value := p.input[functionStart:p.peek]
		nodeType := PlainText

		return NewParentNode(&nodeType,
			NewLeafNode(&value, &nodeType, nil))
	}

	offset := 0
	if *nodeType == Paragraph {
		offset++
	}
	if start < p.current {
		value := p.input[start : p.current+offset]
		nodeType := PlainText
		children = append(children, NewLeafNode(&value, &nodeType, nil))
	}

	p.readX(identSize)

	return NewParentNode(nodeType, children...)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseBold() HtmlNode {
	identSize := 2

	breakCondition := func() bool {
		return p.ch == '*' && p.peekCharX(PeekOnce) == '*'
	}

	childType := Bold
	return p.parseChildren(&identSize, breakCondition, &childType)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseCode() HtmlNode {
	identSize := 3

	breakCondition := func() bool {
		return p.ch == '`' && p.peekCharX(PeekOnce) == '`' && p.peekCharX(PeekTwice) == '`'
	}

	childType := Code
	return p.parseChildren(&identSize, breakCondition, &childType)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseEscaped() HtmlNode {
	p.readChar()

	value := string(p.input[p.current])
	nodeType := PlainText
	newLeaf := NewLeafNode(&value, &nodeType, nil)

	p.readChar()

	return newLeaf
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseItalic() HtmlNode {
	identSize := 1

	breakCondition := func() bool {
		return p.ch == '*' && p.peekCharX(PeekOnce) != '*'
	}

	childType := Italic
	return p.parseChildren(&identSize, breakCondition, &childType)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseImageLink(nodeType *NodeType) HtmlNode {
	functionStart := p.current
	endReads := 1
	childType := PlainText

	if *nodeType == Image {
		p.readChar()
	}

	breakConditionPrefix := func() bool {
		return p.ch == ']'
	}
	prefixNode := p.parseChildren(&endReads, breakConditionPrefix, &childType)
	if p.ch != '(' {
		value := p.input[functionStart:p.current]
		nodeType := PlainText
		return NewLeafNode(&value, &nodeType, nil)
	}

	breakCondtionSrc := func() bool {
		return p.ch == ')'
	}

	srcNode := p.parseChildren(&endReads, breakCondtionSrc, &childType)

	textNode := p.imageLinkChecks(&functionStart)
	if textNode != nil {
		return textNode
	}

	switch *nodeType {
	case Image:
		value := ""
		nodeType := Image
		properties := map[string]string{
			"alt": prefixNode.toHtml(),
			"src": srcNode.toHtml(),
		}
		return NewLeafNode(&value, &nodeType, &properties)

	case Link:
		value := prefixNode.toHtml()
		nodeType := Link
		properties := map[string]string{
			"href": srcNode.toHtml(),
		}
		return NewLeafNode(&value, &nodeType, &properties)

	default:
		return nil
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseOrderedList() HtmlNode {
	consumeIdent := func() {
		for p.isDigit(&p.ch) {
			p.readChar()
		}
		remainingIdent := 2
		p.readX(&remainingIdent)
	}
	consumeIdent()

	breakCondition := func() bool {
		return p.isOrderedIdent()
	}

	endReads := 0
	childType := ListElement
	return p.parseChildren(&endReads, breakCondition, &childType)
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) parseUnorderedList() HtmlNode {
	identSize := 2
	p.readX(&identSize)

	breakCondition := func() bool {
		return p.ch == '-' && p.peekCharX(PeekOnce) == ' '
	}

	endReads := 0
	childType := ListElement
	return p.parseChildren(&endReads, breakCondition, &childType)
}

// ----------------------------------------------------------------------------------------------------------------
// Helper functions.
// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) blockType() NodeType {
	switch p.ch {
	case '>':
		if p.peekCharX(PeekOnce) == ' ' {
			return Quote
		}
	case '-':
		if p.peekCharX(PeekOnce) == ' ' {
			return UnorderedList
		}
	case '#':
		return p.headingType()
	default:
		if p.isDigit(&p.ch) {
			if p.isOrderedIdent() {
				return OrderedList
			}
		}
	}

	return Paragraph
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
	substring := p.input[*start:p.current]
	nodeType := PlainText

	ident := p.isIdent()
	if ident == Escaped {
		if *start < p.current {
			*targetSlice = append(*targetSlice, NewLeafNode(&substring, &nodeType, nil))
		}

		p.readChar()

		escapedChar := string(p.input[p.current])
		*targetSlice = append(*targetSlice, NewLeafNode(&escapedChar, &nodeType, nil))

		p.readChar()

		*start = p.current

	} else if ident != PlainText {
		parseFunc := p.parsingFuncs[ident]

		if *start < p.current {
			*targetSlice = append(*targetSlice, NewLeafNode(&substring, &nodeType, nil))
		}
		*targetSlice = append(*targetSlice, parseFunc())
		*start = p.current

	} else {
		p.readChar()
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) checkBoldNode() bool {
	return p.ch == '*' && p.peekCharX(PeekOnce) == '*' && p.peekCharX(PeekTwice) != '*'
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) checkUnterminatedExceptions(nodeType *NodeType) bool {
	case1 := p.ch == EOF
	case2 := *nodeType != Quote
	case3 := *nodeType != ListElement
	case4 := func() bool {
		switch *nodeType {
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
		offset := 2
		p.readX(&offset)
	case Heading1:
		offset := 2
		p.readX(&offset)
	case Heading2:
		offset := 3
		p.readX(&offset)
	case Heading3:
		offset := 4
		p.readX(&offset)
	case Heading4:
		offset := 5
		p.readX(&offset)
	case Heading5:
		offset := 6
		p.readX(&offset)
	case Heading6:
		offset := 7
		p.readX(&offset)
	default:
		return
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) headingType() NodeType {
	index := 0

	for {
		current := p.readFromIndex(&index)
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

func (p *Parser) imageLinkChecks(functionStart *int) HtmlNode {
	subString := p.input[*functionStart:p.peek]
	nodeType := PlainText

	if p.ch == EOF {
		if p.peekPreviousX(PeekOnce) != ')' {
			return NewLeafNode(&subString, &nodeType, nil)
		}
	} else {
		if p.peekPreviousX(PeekTwice) != ')' {
			return NewLeafNode(&subString, &nodeType, nil)
		}
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) isDigit(ch *rune) bool {
	return *ch >= '0' && *ch <= '9'
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) isIdent() NodeType {
	switch p.ch {
	case '*':
		if p.peekCharX(PeekOnce) == '*' {
			return Bold
		}
		return Italic
	case '`':
		if p.peekCharX(PeekOnce) == '`' && p.peekCharX(PeekTwice) == '`' {
			return Code
		}
	case '!':
		if p.peekCharX(PeekOnce) == '[' {
			return Image
		}
	case '[':
		return Link
	case '-':
		if p.peekCharX(PeekOnce) == ' ' {
			return UnorderedList
		}
	case '\\':
		return Escaped
	default:
		if p.isDigit(&p.ch) {
			if p.isOrderedIdent() {
				return OrderedList
			}
		}
	}
	return PlainText
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) isOrderedIdent() bool {
	index := p.current

	for {
		char := p.readFromIndex(&index)
		if !p.isDigit(&char) {
			break
		}
		index++
	}

	next := index + 1
	if p.readFromIndex(&index) == '.' && p.readFromIndex(&next) == ' ' {
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

func (p *Parser) readFromIndex(index *int) rune {
	if *index >= len(p.input) {
		return EOF
	}
	return rune(p.input[*index])
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) readX(total *int) {
	for range *total {
		p.readChar()
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (p *Parser) registerFunc(nodeType NodeType, fn ParseFunc) {
	p.parsingFuncs[nodeType] = fn
}

// ----------------------------------------------------------------------------------------------------------------
