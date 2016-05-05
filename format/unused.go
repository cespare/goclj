package format

import (
	"strings"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

type symbolCache struct {
	imports  map[string]struct{} // packages appearing in :imports
	symbols  map[string]struct{} // symbols without a / in them; e.g., foo
	prefixes map[string]struct{} // symbol prefixes; e.g., a/foo -> a
}

func findSymbols(roots []parse.Node) *symbolCache {
	syms := &symbolCache{
		imports:  make(map[string]struct{}),
		symbols:  make(map[string]struct{}),
		prefixes: make(map[string]struct{}),
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
			syms.prefixes[name[:i]] = struct{}{}
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

func (sc *symbolCache) hasSym(name string) bool {
	_, ok := sc.symbols[name]
	return ok
}

func (sc *symbolCache) hasAs(name string) bool {
	_, ok := sc.prefixes[name]
	return ok
}

func (sc *symbolCache) hasRequireAsImport(name string) bool {
	name = strings.Replace(name, "-", "_", -1)
	_, ok := sc.imports[name]
	return ok
}

// unused removes unused :as and :refer aliases from r,
// and also returns whether the require is no longer needed at all.
func (sc *symbolCache) unused(r *require) bool {
	if len(r.as) == 0 && r.origRefer == nil && len(r.refer) == 0 {
		// For requires like [foo], which are presumably to load Java
		// classes, don't try to figure out if they're used.
		return false
	}
	for as := range r.as {
		if !sc.hasAs(as) {
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
			if !sc.hasSym(n.Val) {
				r.extractOrigRefer()
				break
			}
		}
	}
	for ref := range r.refer {
		if !sc.hasSym(ref) {
			delete(r.refer, ref)
		}
	}
	return len(r.as) == 0 &&
		!r.referAll &&
		r.origRefer == nil && len(r.refer) == 0 &&
		!sc.hasRequireAsImport(r.name)
}
