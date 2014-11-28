package parse

import (
	"bufio"
	"bytes"
	"fmt"
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

func (t *Tree) Parse() (err error) {
	defer t.recover(&err)
	for {
		node := t.parse()
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

func ParseFile(filename string) (*Tree, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	t := &Tree{
		includeNonSemantic: false,
		lex:                lex(filename, bufio.NewReader(f)),
	}
	if err := t.Parse(); err != nil {
		return nil, err
	}
	return t, nil
}

// parse parses the next top-level item from the token stream. It returns nil if there are no non-EOF tokens
// left in the stream.
func (t *Tree) parse() Node {
	for {
		switch tok := t.next(); tok.typ {
		case tokSymbol:
			switch val := tok.val; val {
			case "nil":
				return &NilNode{NodeNil, tok.pos}
			case "true", "false":
				return &BoolNode{NodeBool, tok.pos, val == "true"}
			default:
				return &SymbolNode{NodeSymbol, tok.pos, tok.val}
			}
		case tokCharLiteral:
			return t.parseCharLiteral(tok)
		case tokComment:
			return &CommentNode{NodeComment, tok.pos, tok.val}
		case tokAtSign:
			return &DerefNode{NodeDeref, tok.pos, t.parse()}
		case tokKeyword:
			return &KeywordNode{NodeKeyword, tok.pos, tok.val}
		case tokLeftParen:
			return t.parseList(tok)
		case tokLeftBrace:
			return t.parseMap(tok)
		case tokCircumflex:
			return &MetadataNode{NodeMetadata, tok.pos, t.parse()}
		case tokNewline:
			return &NewlineNode{NodeNewline, tok.pos}
		case tokNumber:
			// TODO: need to parse the number here; a number token may not be valid.
			return &NumberNode{NodeNumber, tok.pos, tok.val}
		case tokApostrophe:
			return &QuoteNode{NodeQuote, tok.pos, t.parse()}
		case tokString:
			return &StringNode{NodeString, tok.pos, tok.val[1 : len(tok.val)-1]}
		case tokBacktick:
			return &SyntaxQuoteNode{NodeSyntaxQuote, tok.pos, t.parse()}
		case tokTilde:
			next := t.next()
			switch next.typ {
			case tokAtSign:
				return &UnquoteSpliceNode{NodeUnquoteSplice, tok.pos, t.parse()}
			case tokEOF:
				t.unexpectedEOF(next)
			}
			t.backup()
			return &UnquoteNode{NodeUnquote, tok.pos, t.parse()}
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
	return &CharacterNode{NodeCharacter, tok.pos, r}
}

func (t *Tree) parseList(start token) Node {
	var nodes []Node
	for {
		switch tok := t.next(); tok.typ {
		case tokRightParen:
			return &ListNode{NodeList, start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parse()
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
			return &MapNode{NodeMap, start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parse()
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
			return &VectorNode{NodeVector, start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parse()
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
		return t.parseIgnoreForm(tok)
	case `#"`:
		return t.parseRegex(tok)
	case "#{":
		return t.parseSet(tok)
	case "#'":
		return t.parseVarQuote(tok)
	default:
		t.unexpected(tok)
	}
	panic("unreached")
}

func (t *Tree) parseTag(start token) Node {
	tok := t.next()
	switch tok.typ {
	case tokSymbol:
		return &TagNode{NodeTag, start.pos, tok.val}
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
			return &FnLiteralNode{NodeFnLiteral, start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parse()
		if t.includeNonSemantic || isSemantic(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseIgnoreForm(start token) Node {
	tok := t.next()
	if tok.typ != tokSymbol || tok.val != "_" {
		panic("should not happen")
	}
	tok = t.next()
	if tok.typ == tokEOF {
		t.unexpectedEOF(tok)
	}
	t.backup()
	return &IgnoreFormNode{NodeIgnoreForm, start.pos, t.parse()}
}

func (t *Tree) parseRegex(start token) Node {
	tok := t.next()
	if tok.typ != tokString {
		panic("should not happen")
	}
	return &RegexNode{NodeRegex, start.pos, tok.val[1 : len(tok.val)-1]}
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
			return &SetNode{NodeSet, start.pos, nodes}
		case tokEOF:
			t.unexpectedEOF(tok)
		}
		t.backup()
		node := t.parse()
		if t.includeNonSemantic || isSemantic(node) {
			nodes = append(nodes, node)
		}
	}
}

func (t *Tree) parseVarQuote(start token) Node {
	tok := t.next()
	if tok.typ != tokApostrophe {
		panic("should not happen")
	}
	switch tok = t.next(); tok.typ {
	case tokSymbol:
		return &VarQuoteNode{NodeVarQuote, start.pos, tok.val}
	case tokEOF:
		t.unexpectedEOF(tok)
	default:
		t.unexpected(tok)
	}
	panic("unreached")
}
