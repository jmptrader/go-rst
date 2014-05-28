// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.

package parse

import (
	"code.google.com/p/go.text/unicode/norm"
	"fmt"
	"github.com/demizer/go-elog"
	"github.com/demizer/go-spew/spew"
	"reflect"
)

var spd = spew.ConfigState{Indent: "\t", DisableMethods: true}

type systemMessageLevel int

const (
	levelInfo systemMessageLevel = iota
	levelWarning
	levelError
	levelSevere
)

var systemMessageLevels = [...]string{
	"INFO",
	"WARNING",
	"ERROR",
	"SEVERE",
}

func (s systemMessageLevel) String() string {
	return systemMessageLevels[s]
}

type parserMessage int

const (
	warningShortUnderline parserMessage = iota
	errorUnexpectedSectionTitle
	errorUnexpectedSectionTitleOrTransition
)

var parserErrors = [...]string{
	"warningShortUnderline",
	"errorUnexpectedSectionTitle",
	"errorUnexpectedSectionTitleOrTransition",
}

func (p parserMessage) String() string {
	return parserErrors[p]
}

func (p parserMessage) Message() (s string) {
	switch p {
	case warningShortUnderline:
		s = "Title underline too short."
	case errorUnexpectedSectionTitle:
		s = "Unexpected section title."
	case errorUnexpectedSectionTitleOrTransition:
		s = "Unexpected section title or transition."
	}
	return
}

func (p parserMessage) Level() (s systemMessageLevel) {
	switch p {
	case warningShortUnderline:
		s = levelWarning
	case errorUnexpectedSectionTitle:
		s = levelSevere
	case errorUnexpectedSectionTitleOrTransition:
		s = levelSevere
	}
	return
}

type sectionLevels []*SectionNode

func (s *sectionLevels) String() string {
	var out string
	for _, sec := range *s {
		out += fmt.Sprintf("level: %d, rune: %q, overline: %t, length: %d\n",
			sec.Level, sec.UnderLine.Rune, sec.OverLine != nil, sec.Length)
	}
	return out
}

// Returns nil if not found
func (s *sectionLevels) FindByRune(adornChar rune) *SectionNode {
	for _, sec := range *s {
		if sec.UnderLine.Rune == adornChar {
			return sec
		}
	}
	return nil
}

// If exists == true, a section node with the same text and underline has been found in
// sectionLevels, sec is the matching SectionNode. If exists == false, then the sec return value is
// the similarly leveled SectionNode. If exists == false and sec == nil, then the SectionNode added
// to sectionLevels is a new Node.
func (s *sectionLevels) Add(section *SectionNode) (exists bool, sec *SectionNode) {
	sec = s.FindByRune(section.UnderLine.Rune)
	if sec != nil {
		if sec.Text == section.Text {
			return true, sec
		} else if sec.Text != section.Text {
			section.Level = sec.Level
		}
	} else {
		section.Level = len(*s) + 1
	}
	exists = false
	*s = append(*s, section)
	return
}

func (s *sectionLevels) Level() int {
	return len(*s)
}

// Parse is the entry point for the reStructuredText parser.
func Parse(name, text string) (t *Tree, errors []error) {
	t = New(name)
	if !norm.NFC.IsNormalString(text) {
		text = norm.NFC.String(text)
	}
	t.text = text
	_, errors = t.Parse(text, t)
	return
}

func New(name string) *Tree {
	return &Tree{
		Name:          name,
		Nodes:         newList(),
		nodeTarget:    newList(),
		sectionLevels: new(sectionLevels),
		indentWidth:   indentWidth,
	}
}

const (
	tokenZero   = 3
	indentWidth = 4 // Default indent width
)

type Tree struct {
	Name             string
	Nodes            *NodeList // The root node list
	nodeTarget       *NodeList // Used by the parser to add nodes to a target NodeList
	Errors           []error
	text             string
	lex              *lexer
	tokenBackupCount int
	tokenPeekCount   int
	token            [7]*item
	sectionLevels    *sectionLevels // Encountered section levels
	id               int            // The unique id of the node in the tree
	indentWidth      int
	indentLevel      int
}

func (t *Tree) errorf(format string, args ...interface{}) {
	format = fmt.Sprintf("go-rst: %s:%d: %s\n", t.Name, t.lex.lineNumber(), format)
	t.Errors = append(t.Errors, fmt.Errorf(format, args...))
}

func (t *Tree) error(err error) {
	t.errorf("%s\n", err)
}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(lex *lexer) {
	t.lex = lex
}

// stopParse terminates parsing.
func (t *Tree) stopParse() {
	t.Nodes = nil
	t.nodeTarget = nil
	t.lex = nil
}

func (t *Tree) Parse(text string, treeSet *Tree) (tree *Tree, errors []error) {
	log.Debugln("Start")
	t.startParse(lex(t.Name, text))
	t.text = text
	t.parse(treeSet)
	log.Debugln("End")
	return t, t.Errors
}

func (t *Tree) parse(tree *Tree) {
	log.Debugln("Start")

	t.nodeTarget = t.Nodes

	for t.peek(1).Type != itemEOF {
		var n Node

		token := t.next()
		log.Infof("\nParser got token: %#+v\n\n", token)

		switch token.Type {
		case itemSectionAdornment:
			n = t.section(token)
		case itemParagraph:
			n = newParagraph(token, &t.id)
		case itemSpace:
			n = t.indent(token)
			if n == nil {
				continue
			}
		case itemTitle, itemBlankLine:
			// itemTitle is consumed when evaluating itemSectionAdornment
			continue
		case itemEOF:
			goto exit
		default:
			t.errorf("%q Not implemented!", token.Type)
			continue
		}

		t.nodeTarget.append(n)
		switch n.NodeType() {
		case NodeSection, NodeBlockQuote:
			// Set the loop to append items to the NodeList of the new section
			t.nodeTarget = reflect.ValueOf(n).Elem().FieldByName("NodeList").Addr().Interface().(*NodeList)
		}
	}

	exit:
	log.Debugln("End")
}

func (t *Tree) backup() *item {
	t.tokenBackupCount++
	// log.Debugln("t.tokenBackupCount:", t.tokenPeekCount)
	for i := len(t.token) - 1; i > 0; i-- {
		t.token[i] = t.token[i-1]
		t.token[i-1] = nil
	}
	// log.Debugf("\n##### backup() aftermath #####\n\n")
	// spd.Dump(t.token)
	return t.token[tokenZero-t.tokenBackupCount]
}

func (t *Tree) peekBack(pos int) *item {
	return t.token[tokenZero-pos]
}

func (t *Tree) peek(pos int) *item {
	// log.Debugln("t.tokenPeekCount:", t.tokenPeekCount, "Pos:", pos)
	if pos < 1 {
		panic("pos cannot be < 1")
	}
	var nItem *item
	for i := 0; i < pos; i++ {
		// log.Debugln("i:", i, "peekCount:", t.tokenPeekCount, "pos:", pos)
		if t.tokenPeekCount > i {
			nItem = t.token[tokenZero+i]
			log.Debugf("Using %#+v\n", nItem)
			continue
		}
		log.Debugln(tokenZero + t.tokenPeekCount + i)
		if t.token[tokenZero + t.tokenPeekCount + i + 1] == nil {
			t.tokenPeekCount++
			// log.Debugln("Getting next item")
			t.token[tokenZero+t.tokenPeekCount+i] = t.lex.nextItem()
			nItem = t.token[tokenZero+t.tokenPeekCount+i]
		} else {
			nItem = t.token[tokenZero+t.tokenPeekCount+i]
		}
	}
	// log.Debugf("\n##### peek() aftermath #####\n\n")
	// spd.Dump(t.token)
	// log.Debugf("Returning: %#+v\n", nItem)
	return nItem
}

// skip shifts the pointers left in t.token, pos is the amount to shift
func (t *Tree) skip(num int) {
	for i := num; i > 0; i-- {
		for x := 0; x < len(t.token)-1; x++ {
			t.token[x] = t.token[x+1]
			t.token[x+1] = nil
		}
	}
}

func (t *Tree) next() *item {
	// log.Debugln("t.tokenPeekCount:", t.tokenPeekCount)
	if t.tokenPeekCount > 0 {
		t.skip(t.tokenPeekCount)
	} else {
		t.skip(1)
		t.token[tokenZero] = t.lex.nextItem()
	}
	t.tokenBackupCount, t.tokenPeekCount = 0, 0
	// log.Debugf("\n##### next() aftermath #####\n\n")
	// spd.Dump(t.token)
	return t.token[tokenZero]
}

func (t *Tree) section(i *item) Node {
	log.Debugln("Start")
	var overAdorn, title, underAdorn *item
	var overline bool
	var sysMessage Node

	peekBack := t.peekBack(1)
	if peekBack != nil {
		if peekBack.Type == itemSpace {
			// Looking back past the white space
			if t.peekBack(2).Type == itemTitle {
				return t.systemMessage(errorUnexpectedSectionTitle)
			}
			return t.systemMessage(errorUnexpectedSectionTitleOrTransition)
		} else if peekBack.Type == itemTitle {
			if t.peekBack(2) != nil && t.peekBack(2).Type == itemSectionAdornment {
				// The overline of the section
				overline = true
				overAdorn = peekBack
			}
		}
	}

	title = t.peekBack(1)
	underAdorn = i

	// TODO: Change these into proper error messages!
	// Check adornment for proper syntax
	if underAdorn.Type == itemSpace {
		t.backup() // Put the parser back on the title
		return t.systemMessage(errorUnexpectedSectionTitle)
	} else if overline && title.Length != overAdorn.Length {
		t.errorf("Section over line not equal to title length!")
	} else if overline && overAdorn.Text != underAdorn.Text {
		t.errorf("Section title over line does not match section title under line.")
	}

	sec := newSection(title, overAdorn, underAdorn, &t.id)
	exists, eSec := t.sectionLevels.Add(sec)
	if exists && eSec != nil {
		t.errorf("SectionNode using Text \"%s\" and Rune '%s' was previously parsed!",
			sec.Text, string(sec.UnderLine.Rune))
	} else if !exists && eSec != nil {
		// There is a matching level in sectionLevels
		t.nodeTarget = &(*t.sectionLevels)[sec.Level-2].NodeList
	}

	// System messages have to be applied after the section is created in order to preserve
	// a consecutive id number.
	if title.Length != underAdorn.Length {
		sysMessage = t.systemMessage(warningShortUnderline)
		sec.NodeList = append(sec.NodeList, sysMessage)
	}

	log.Debugln("End")
	return sec
}

func (t *Tree) systemMessage(err parserMessage) Node {
	var lbText string
	var lbTextLen int
	var backToken int

	s := newSystemMessage(&item{
		Type: itemSystemMessage,
		Line: t.token[tokenZero].Line,
	},
		err.Level(), &t.id)

	msg := newParagraph(&item{
		Text:   err.Message(),
		Length: len(err.Message()),
	}, &t.id)

	switch err {
	case warningShortUnderline, errorUnexpectedSectionTitle:
		log.Debugln("FOUND", err)
		backToken = tokenZero - 1
		if t.peekBack(1).Type == itemSpace {
			backToken = tokenZero - 2
		}
		lbText = t.token[backToken].Text.(string) + "\n" + t.token[tokenZero].Text.(string)
		lbTextLen = len(lbText) + 1
	case errorUnexpectedSectionTitleOrTransition:
		log.Debugln("FOUND errorUnexpectedSectionTitleOrTransition")
		lbText = t.token[tokenZero].Text.(string)
		lbTextLen = len(lbText)
	}

	lb := newLiteralBlock(&item{
		Type:   itemLiteralBlock,
		Text:   lbText,
		Length: lbTextLen, // Add one to account for the backslash
	}, &t.id)

	s.NodeList = append(s.NodeList, msg, lb)
	return s
}

func (t *Tree) indent(i *item) Node {
	level := i.Length / t.indentWidth
	if t.peekBack(1).Type == itemBlankLine {
		if t.indentLevel == level {
			// Append to the current blockquote NodeList
			return nil
		}
		t.indentLevel = level
		return newBlockQuote(&item{Type: itemBlockquote, Line: i.Line}, level, &t.id)
	}
	return nil
}
