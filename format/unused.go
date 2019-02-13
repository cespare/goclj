package format

import (
	"strings"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

type symbolCache struct {
	imports    map[string]struct{} // packages appearing in :imports
	symbols    map[string]struct{} // symbols without a / in them; e.g., foo
	namespaces map[string]struct{} // symbol namespaces; e.g., a/foo -> a
}

func findSymbols(roots []parse.Node) *symbolCache {
	syms := &symbolCache{
		imports:    make(map[string]struct{}),
		symbols:    make(map[string]struct{}),
		namespaces: make(map[string]struct{}),
	}
	var find func(n parse.Node)
	find = func(n parse.Node) {
		var name string
		switch n := n.(type) {
		case *parse.SymbolNode:
			name = n.Val
		case *parse.VarQuoteNode:
			name = n.Val
		default:
			for _, child := range n.Children() {
				find(child)
			}
			return
		}
		i := strings.IndexRune(name, '/')
		if i < 0 {
			syms.symbols[name] = struct{}{}
		} else {
			syms.namespaces[name[:i]] = struct{}{}
		}
	}
	for _, root := range roots {
		if goclj.FnFormSymbol(root, "ns") {
			for _, n := range root.Children()[1:] {
				if goclj.FnFormKeyword(n, ":import") {
					for _, n1 := range n.Children()[1:] {
						syms.findImports(n1)
					}
				}
			}
		} else {
			find(root)
		}
	}
	return syms
}

func (sc *symbolCache) findImports(n parse.Node) {
	switch n := n.(type) {
	case *parse.SymbolNode:
		if i := strings.LastIndexByte(n.Val, '.'); i >= 0 {
			sc.imports[n.Val[:i]] = struct{}{}
		}
	case *parse.ListNode, *parse.VectorNode:
		nodes := n.Children()
		if len(nodes) > 0 {
			if sym, ok := nodes[0].(*parse.SymbolNode); ok {
				sc.imports[sym.Val] = struct{}{}
			}
		}
	}
}

func (sc *symbolCache) usesSym(name string) bool {
	_, ok := sc.symbols[name]
	return ok
}

func (sc *symbolCache) usesNamespace(name string) bool {
	_, ok := sc.namespaces[name]
	return ok
}

func (sc *symbolCache) usesRequireAsImport(name string) bool {
	name = strings.Replace(name, "-", "_", -1)
	_, ok := sc.imports[name]
	return ok
}

// unused removes unused :as and :refer aliases from r,
// and also returns whether the require is no longer needed at all.
func (sc *symbolCache) unused(r *require) bool {
	for as := range r.as {
		if !sc.usesNamespace(as) {
			delete(r.as, as)
		}
	}
	if r.origRefer != nil {
		// If origRefer doesn't have any unused elements, leave it
		// alone. Otherwise, rewrite it as a refer and handle below.
		for _, n := range r.origRefer {
			n, ok := n.(*parse.SymbolNode)
			if !ok {
				continue
			}
			if !sc.usesSym(n.Val) {
				r.extractOrigRefer()
				break
			}
		}
	}
	for ref := range r.refer {
		if !sc.usesSym(ref) {
			delete(r.refer, ref)
		}
	}
	return !sc.usesNamespace(r.name) &&
		!sc.usesRequireAsImport(r.name) &&
		len(r.as) == 0 &&
		!r.referAll &&
		r.origRefer == nil && len(r.refer) == 0
}
