package format

import (
	"strings"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

type symbolCache struct {
	symbols  map[string]struct{} // symbols without a / in them; e.g., foo
	prefixes map[string]struct{} // symbol prefixes; e.g., a/foo -> a
}

func findSymbols(roots []parse.Node) *symbolCache {
	syms := &symbolCache{
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
		if !goclj.FnFormSymbol(root, "ns") {
			find(root)
		}
	}
	return syms
}

func (sc *symbolCache) hasSym(name string) bool {
	_, ok := sc.symbols[name]
	return ok
}

func (sc *symbolCache) hasAs(name string) bool {
	_, ok := sc.prefixes[name]
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
		r.origRefer == nil && len(r.refer) == 0
}
