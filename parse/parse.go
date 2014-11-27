package parse

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
)

type Tree struct {
	Roots []Node

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
		t.Roots = append(t.Roots, node)
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

func (t *Tree) unexpected(pos *Pos) { t.errorf(pos, "unexpected token") }

func ParseFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	t := &Tree{lex: lex(filename, bufio.NewReader(f))}
	if err := t.Parse(); err != nil {
		log.Fatal(err)
	}
	fmt.Print(t.String())
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
			// TODO: option to save
			continue
		case tokAtSign:
			return &DerefNode{NodeDeref, tok.pos, t.parse()}
		case tokLeftParen:
			return t.parseList(tok)
		case tokApostrophe:
			return &QuoteNode{NodeQuote, tok.pos, t.parse()}
		case tokBacktick:
			return &SyntaxQuoteNode{NodeSyntaxQuote, tok.pos, t.parse()}
		case tokTilde:
			next := t.next()
			if next.typ == tokAtSign {
				return &UnquoteSpliceNode{NodeUnquoteSplice, tok.pos, t.parse()}
			}
			t.backup()
			return &UnquoteNode{NodeUnquote, tok.pos, t.parse()}
		case tokEOF:
			return nil
		default:
			t.unexpected(tok.pos)
		}
		panic("unreached")
	}
}

func (t *Tree) parseCharLiteral(tok token) Node {
	var r rune
	switch tok.val {
	case `\newline`:
		r = '\n'
	default:
		t.unexpected(tok.pos)
	}
	return &CharacterNode{NodeCharacter, tok.pos, r}
}

func (t *Tree) parseList(start token) Node {
	var nodes []Node
	for {
		if tok := t.next(); tok.typ == tokRightParen {
			return &ListNode{NodeList, start.pos, nodes}
		}
		t.backup()
		nodes = append(nodes, t.parse())
	}
}
