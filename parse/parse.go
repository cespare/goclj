package parse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Tree struct {
	Roots []Node

	// Config
	includeNonSemantic  bool
	ignoreCommentForm   bool
	ignoreReaderDiscard bool

	// Parser state
	tok       token // single-item lookahead
	peekCount int
	lex       *lexer
	inLambda  bool
}

// String pretty-prints the tree recursively using each Node's String().
func (t *Tree) String() string { return nodesToString(t.Roots, 0) }

func nodesToString(nodes []Node, depth int) string {
	var buf bytes.Buffer
	for _, node := range nodes {
		buf.WriteString(strings.Repeat("  ", depth))
		buf.WriteString(node.String())
		buf.WriteString("\n")
		buf.WriteString(nodesToString(node.Children(), depth+1))
	}
	return buf.String()
}

func (t *Tree) parse() (err error) {
	defer t.recover(&err)
	var linkParents func(Node)
	linkParents = func(n Node) {
		for _, c := range n.Children() {
			c.SetParent(n)
			linkParents(c)
		}
	}
	for {
		node := t.parseNext()
		if node == nil {
			break
		}
		linkParents(node)
		if t.includeNode(node) {
			t.Roots = append(t.Roots, node)
		}
	}
	return nil
}

type lexError struct{ err error }
type parseError struct{ err error }

func (t *Tree) recover(err *error) {
	if e := recover(); e != nil {
		switch e := e.(type) {
		case lexError:
			*err = e.err
		case parseError:
			*err = e.err
		default:
			panic(e)
		}
	}
}

func (t *Tree) nextToken() token {
	tok := t.lex.nextToken()
	if tok.typ == tokError {
		panic(lexError{tok.AsError()})
	}
	return tok
}

func (t *Tree) next() token {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.tok = t.nextToken()
	}
	return t.tok
}

func (t *Tree) backup() {
	t.peekCount++
	if t.peekCount > 1 {
		panic("backup() called twice consecutively")
	}
}

func (t *Tree) peek() token {
	if t.peekCount == 0 {
		t.peekCount++
		t.tok = t.nextToken()
	}
	return t.tok
}

func (t *Tree) errorf(pos *Pos, format string, args ...interface{}) {
	panic(parseError{pos.FormatError("parse", fmt.Sprintf(format, args...))})
}

func (t *Tree) unexpected(tok token) { t.errorf(tok.pos, "unexpected token %q", tok.val) }

func (t *Tree) unexpectedEOF(tok token) { t.errorf(tok.pos, "unexpected EOF") }

// ParseOpts is a bitset of parsing options for Reader and File.
type ParseOpts uint

const (
	// IncludeNonSemantic makes the parser include non-semantic nodes:
	// CommentNodes and NewlineNodes.
	IncludeNonSemantic ParseOpts = 1 << iota
	// IgnoreCommentForm makes the parser ignore (comment ...) forms.
	IgnoreCommentForm
	// IgnoreReaderDiscard makes the parser ignore forms preceded by #_.
	IgnoreReaderDiscard
)

func Reader(r io.Reader, filename string, opts ParseOpts) (*Tree, error) {
	t := &Tree{
		includeNonSemantic:  opts&IncludeNonSemantic != 0,
		ignoreCommentForm:   opts&IgnoreCommentForm != 0,
		ignoreReaderDiscard: opts&IgnoreReaderDiscard != 0,
		lex:                 lex(filename, bufio.NewReader(r)),
	}
	if err := t.parse(); err != nil {
		return nil, err
	}
	return t, nil
}

func File(filename string, opts ParseOpts) (*Tree, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Reader(f, filename, opts)
}

// parseNext parses the next top-level item from the token stream.
// It returns nil if there are no non-EOF tokens left in the stream.
func (t *Tree) parseNext() Node {
	for {
		switch tok := t.next(); tok.typ {
		case tokSymbol:
			switch val := tok.val; val {
			case "nil":
				return &NilNode{Pos: tok.pos}
			case "true", "false":
				return &BoolNode{Pos: tok.pos, Val: val == "true"}
			default:
				return &SymbolNode{Pos: tok.pos, Val: tok.val}
			}
		case tokCharLiteral:
			return t.parseCharLiteral(tok)
		case tokComment:
			return &CommentNode{Pos: tok.pos, Text: tok.val}
		case tokAtSign:
			return &DerefNode{Pos: tok.pos, Node: t.parseNextSemantic()}
		case tokKeyword:
			return &KeywordNode{Pos: tok.pos, Val: tok.val}
		case tokLeftParen:
			return t.parseList(tok)
		case tokLeftBrace:
			return t.parseMap(tok)
		case tokCircumflex:
			return t.parseMetadata(tok)
		case tokNewline:
			return &NewlineNode{Pos: tok.pos}
		case tokNumber:
			// TODO: need to parse the number here; a number token may not be valid.
			return &NumberNode{Pos: tok.pos, Val: tok.val}
		case tokApostrophe:
			return &QuoteNode{Pos: tok.pos, Node: t.parseNextSemantic()}
		case tokString:
			return &StringNode{Pos: tok.pos, Val: tok.val[1 : len(tok.val)-1]}
		case tokBacktick:
			return &SyntaxQuoteNode{Pos: tok.pos, Node: t.parseNextSemantic()}
		case tokTilde:
			next := t.next()
			switch next.typ {
			case tokAtSign:
				return &UnquoteSpliceNode{Pos: tok.pos, Node: t.parseNextSemantic()}
			case tokEOF:
				t.unexpectedEOF(next)
			}
			t.backup()
			return &UnquoteNode{Pos: tok.pos, Node: t.parseNextSemantic()}
		case tokLeftBracket:
			return t.parseVector(tok)
		case tokDispatch:
			return t.parseDispatch(tok)
		case tokOctothorpe:
			return t.parseTag(tok)
		case tokEOF:
			return nil
		default:
			t.unexpected(tok)
		}
		panic("unreached")
	}
}

// parseNextSemantic parses the next top-level semantically meaningful item from
// the token stream. It expects that such an item exists; if it reaches EOF
// before such an item is found, it gives an unexpected EOF error.
func (t *Tree) parseNextSemantic() Node {
	for {
		if next := t.next(); next.typ == tokEOF {
			t.unexpectedEOF(next)
		}
		t.backup()
		n := t.parseNext()
		if isSemantic(n) {
			return n
		}
	}
}

func (t *Tree) parseCharLiteral(tok token) *CharacterNode {
	var r rune
	val := tok.val[1:]
	runes := []rune(val)
	switch len(runes) {
	case 1:
		r = runes[0]
	case 0:
		t.errorf(tok.pos, "invalid character literal")
	default:
		switch val {
		case "newline":
			r = '\n'
		case "space":
			r = ' '
		case "tab":
			r = '\t'
		case "formfeed":
			r = '\f'
		case "backspace":
			r = 0x7f
		case "return":
			r = '\r'
		default:
			switch runes[0] {
			case 'o':
				n, err := strconv.ParseInt(val[1:], 8, 32)
				if len(val) != 4 || err != nil || n < 0 {
					t.errorf(tok.pos, "invalid octal literal")
				}
				r = rune(n)
			case 'u':
				n, err := strconv.ParseInt(val[1:], 16, 32)
				if len(val) != 5 || err != nil || n < 0 {
					t.errorf(tok.pos, "invalid unicode literal")
				}
				r = rune(n)
			default:
				t.errorf(tok.pos, "invalid character literal")
			}
		}
	}
	return &CharacterNode{
		Pos:  tok.pos,
		Val:  r,
		Text: tok.val,
	}
}

func (t *Tree) parseList(start token) *ListNode {
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightParen:
			return &ListNode{Pos: start.pos, Nodes: nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNode(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseMap(start token) *MapNode {
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightBrace:
			return &MapNode{Pos: start.pos, Nodes: nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNode(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseVector(start token) *VectorNode {
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightBracket:
			return &VectorNode{Pos: start.pos, Nodes: nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNode(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseDispatch(tok token) Node {
	switch tok.val {
	case "#(":
		return t.parseFnLiteral(tok)
	case "#?", "#?@":
		return t.parseReaderCond(tok)
	case "#:":
		return t.parseNamespacedMap(tok)
	case "#_":
		return t.parseReaderDiscard(tok)
	case "#=":
		return t.parseReaderEval(tok)
	case `#"`:
		return t.parseRegex(tok)
	case "#{":
		return t.parseSet(tok)
	case "#'":
		return t.parseVarQuote(tok)
	case "#^":
		return t.parseMetadata(tok)
	case "#!":
	case "#<":
	default:
		t.unexpected(tok)
	}
	panic("unreached")
}

func (t *Tree) parseTag(start token) *TagNode {
	tok := t.next()
	switch tok.typ {
	case tokSymbol:
		return &TagNode{Pos: start.pos, Val: tok.val}
	case tokEOF:
		t.unexpectedEOF(tok)
	default:
		t.unexpected(tok)
	}
	panic("not reached")
}

func (t *Tree) parseFnLiteral(start token) *FnLiteralNode {
	if t.inLambda {
		t.errorf(start.pos, "cannot nest fn literals")
	}
	tok := t.next()
	if tok.typ != tokLeftParen {
		panic("should not happen")
	}
	t.inLambda = true
	var nodes []Node
	for {
		switch tok = t.next(); tok.typ {
		case tokRightParen:
			t.inLambda = false
			return &FnLiteralNode{Pos: start.pos, Nodes: nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNode(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseReaderCond(start token) Node {
	tok := t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	if tok.typ != tokLeftParen {
		t.errorf(tok.pos, "reader conditional body must be a list")
	}
	list := t.parseList(tok)
	switch start.val {
	case "#?":
		return &ReaderCondNode{Pos: start.pos, Nodes: list.Nodes}
	case "#?@":
		return &ReaderCondSpliceNode{Pos: start.pos, Nodes: list.Nodes}
	default:
		panic("should not happen")
	}
}

func (t *Tree) parseNamespacedMap(start token) *MapNode {
	tok := t.next()
	if tok.typ != tokKeyword {
		panic("should not happen")
	}
	ns := tok.val
	tok = t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	if tok.typ != tokLeftBrace {
		t.errorf(tok.pos, "namespaced map must have a map")
	}
	m := t.parseMap(tok)
	m.Namespace = ns
	return m
}

func (t *Tree) parseMetadata(start token) *MetadataNode {
	tok := t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	t.backup()
	return &MetadataNode{Pos: start.pos, Node: t.parseNext()}
}

func (t *Tree) parseReaderDiscard(start token) *ReaderDiscardNode {
	tok := t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	t.backup()
	return &ReaderDiscardNode{Pos: start.pos, Node: t.parseNext()}
}

func (t *Tree) parseReaderEval(start token) *ReaderEvalNode {
	tok := t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	t.backup()
	return &ReaderEvalNode{Pos: start.pos, Node: t.parseNext()}
}

func (t *Tree) parseRegex(start token) *RegexNode {
	tok := t.next()
	if tok.typ != tokString {
		panic("should not happen")
	}
	return &RegexNode{Pos: start.pos, Val: tok.val[1 : len(tok.val)-1]}
}

func (t *Tree) parseSet(start token) *SetNode {
	tok := t.next()
	if tok.typ != tokLeftBrace {
		panic("should not happen")
	}
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightBrace:
			return &SetNode{Pos: start.pos, Nodes: nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNode(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseVarQuote(start token) *VarQuoteNode {
	switch tok := t.next(); tok.typ {
	case tokSymbol:
		return &VarQuoteNode{Pos: start.pos, Val: tok.val}
	case tokEOF:
		t.unexpectedEOF(tok)
	default:
		t.unexpected(tok)
	}
	panic("unreached")
}

func (t *Tree) includeNode(node Node) bool {
	if list, ok := node.(*ListNode); ok {
		children := list.Children()
		if len(children) > 0 {
			if sym, ok := children[0].(*SymbolNode); ok {
				if sym.Val == "comment" && t.ignoreCommentForm {
					return false
				}
			}
		}
	}
	if _, ok := node.(*ReaderDiscardNode); ok && t.ignoreReaderDiscard {
		return false
	}
	if !t.includeNonSemantic && !isSemantic(node) {
		return false
	}
	return true
}
