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
	includeNonSemantic bool

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
	for {
		node := t.parseNext()
		if node == nil {
			break
		}
		if t.includeNonSemantic || isSemantic(node) {
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
)

func Reader(r io.Reader, filename string, opts ParseOpts) (*Tree, error) {
	t := &Tree{
		includeNonSemantic: opts&IncludeNonSemantic != 0,
		lex:                lex(filename, bufio.NewReader(r)),
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
				return &NilNode{tok.pos}
			case "true", "false":
				return &BoolNode{tok.pos, val == "true"}
			default:
				return &SymbolNode{tok.pos, tok.val}
			}
		case tokCharLiteral:
			return t.parseCharLiteral(tok)
		case tokComment:
			return &CommentNode{tok.pos, tok.val}
		case tokAtSign:
			return &DerefNode{tok.pos, t.parseNextSemantic()}
		case tokKeyword:
			return &KeywordNode{tok.pos, tok.val}
		case tokLeftParen:
			return t.parseList(tok)
		case tokLeftBrace:
			return t.parseMap(tok)
		case tokCircumflex:
			return t.parseMetadata(tok)
		case tokNewline:
			return &NewlineNode{tok.pos}
		case tokNumber:
			// TODO: need to parse the number here; a number token may not be valid.
			return &NumberNode{tok.pos, tok.val}
		case tokApostrophe:
			return &QuoteNode{tok.pos, t.parseNextSemantic()}
		case tokString:
			return &StringNode{tok.pos, tok.val[1 : len(tok.val)-1]}
		case tokBacktick:
			return &SyntaxQuoteNode{tok.pos, t.parseNextSemantic()}
		case tokTilde:
			next := t.next()
			switch next.typ {
			case tokAtSign:
				return &UnquoteSpliceNode{tok.pos, t.parseNextSemantic()}
			case tokEOF:
				t.unexpectedEOF(next)
			}
			t.backup()
			return &UnquoteNode{tok.pos, t.parseNextSemantic()}
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

func (t *Tree) parseCharLiteral(tok token) Node {
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
	return &CharacterNode{tok.pos, r, tok.val}
}

func (t *Tree) parseList(start token) Node {
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightParen:
			return &ListNode{start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNonSemantic || isSemantic(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseMap(start token) Node {
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightBrace:
			return &MapNode{start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNonSemantic || isSemantic(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseVector(start token) Node {
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightBracket:
			return &VectorNode{start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNonSemantic || isSemantic(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseDispatch(tok token) Node {
	switch tok.val {
	case "#(":
		return t.parseFnLiteral(tok)
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

func (t *Tree) parseTag(start token) Node {
	tok := t.next()
	switch tok.typ {
	case tokSymbol:
		return &TagNode{start.pos, tok.val}
	case tokEOF:
		t.unexpectedEOF(tok)
	default:
		t.unexpected(tok)
	}
	panic("not reached")
}

func (t *Tree) parseFnLiteral(start token) Node {
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
			return &FnLiteralNode{start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNonSemantic || isSemantic(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseMetadata(start token) Node {
	tok := t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	t.backup()
	return &MetadataNode{start.pos, t.parseNext()}
}

func (t *Tree) parseReaderDiscard(start token) Node {
	tok := t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	t.backup()
	return &ReaderDiscardNode{start.pos, t.parseNext()}
}

func (t *Tree) parseReaderEval(start token) Node {
	tok := t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	t.backup()
	return &ReaderEvalNode{start.pos, t.parseNext()}
}

func (t *Tree) parseRegex(start token) Node {
	tok := t.next()
	if tok.typ != tokString {
		panic("should not happen")
	}
	return &RegexNode{start.pos, tok.val[1 : len(tok.val)-1]}
}

func (t *Tree) parseSet(start token) Node {
	tok := t.next()
	if tok.typ != tokLeftBrace {
		panic("should not happen")
	}
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightBrace:
			return &SetNode{start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parseNext()
		if t.includeNonSemantic || isSemantic(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseVarQuote(start token) Node {
	switch tok := t.next(); tok.typ {
	case tokSymbol:
		return &VarQuoteNode{start.pos, tok.val}
	case tokEOF:
		t.unexpectedEOF(tok)
	default:
		t.unexpected(tok)
	}
	panic("unreached")
}
