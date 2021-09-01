# goclj

**Go tools for working with Clojure code.**

[![Go Reference](https://pkg.go.dev/badge/github.com/cespare/goclj.svg)](https://pkg.go.dev/github.com/cespare/goclj)

The parse ([doc](https://pkg.go.dev/github.com/cespare/goclj/parse)) and format
([doc](https://pkg.go.dev/github.com/cespare/goclj/format)) packages implement
Clojure code parsing and (formatted) printing, respectively.

cljfmt is a command-line tool (inspired by gofmt) that uses format to read and
reformat Clojure code. Because it parses the code, its formatting
transformations are safe (they cannot change semantics).

Additionally, cljfmt applies various transformations to the code; these are
discussed in the **Transforms** section, below.

To install or update, run

    go install github.com/cespare/goclj/cljfmt@latest

(You'll need Go 1.16+). Alternatively, you can download a clfjmt binary from the
[Releases page](https://github.com/cespare/goclj/releases).

Here is the output of `cljfmt -h`:

```
usage: cljfmt [flags] [paths...]
Any directories given will be recursively walked. If no paths are provided,
cljfmt reads from standard input.

Flags:
  -c value
        path to config file (default /home/caleb/.cljfmt)
  -disable-transform value
        turn off the named transform (default none)
  -enable-transform value
        turn on the named transform (default none)
  -l    print files whose formatting differs from cljfmt's
  -w    write result to (source) file instead of stdout

See the goclj README for more documentation of the available transforms.
```

## Transforms

Cljfmt can perform many different transformations on the parsed tree before
emitting formatted code. These vary in how aggressive they are and whether they
introduce the possibility of "false positives"; i.e., unwanted changes to
code semantics. The default transformations are very safe. The non-default ones
can be enabled with the `-enable-transform` command-line flag; after running one
of these transformations, you should verify that the code did not break in some
way (typically by running tests).

### sort-import-require (default: on)

Sort :import and :require declarations in ns blocks.

### enforce-ns-style (default: on)

Apply a few common `ns` style rules (based on [How to ns]):

* Clauses use keywords (`:require`) rather than symbols (`require`)
* Clauses are lists (`(:require ...)`) rather than vectors (`[:require ...]`)
* There is a newline after `:require` or `:import`
* For `:require` specifically:
  - Each `require` is written as a vector, not a list
  - Each `require`d namespace is written inside a vector (not a plain symbol)
  - `:refer`ed and `:exclude`d items use vectors, not lists
* For `:import` specifically:
  - Each `import` is written as a list, not a vector
  - Plain symbols become lists (`java.util.Date` becomes `(java.util Date)`)

[How to ns]: https://stuartsierra.com/2016/clojure-how-to-ns.html

### remove-trailing-newlines (default: on)

Remove extra newlines following sequence-like forms, so that parentheses are written on the same
line. For example,

    (foo bar
     )

becomes

    (foo bar)

### fix-defn-arglist-newline (default: on)

Move the arg vector of defns to the same line, if appropriate:

    (defn foo
      [x] ...)

becomes

    (defn foo [x]
      ...)

if there's no newline after the arg list.

### fix-defmethod-dispatch-val-newline (default: on)

Move the dispatch-val of a defmethod to the same line, so that

    (defmethod foo
      :bar
      [x] ...)

becomes

    (defmethod foo :bar
      [x] ...)

### remove-extra-blank-lines (default: on)

Consolidate consecutive blank lines into a single blank line.

### fix-if-newline-consistency (default: on)

Ensure that if one arm of an `if` is on its own line, the other is as well.
Either of these:

    (if foo? a
      b)

    (if foo?
      a b)

become

    (if foo?
      a
      b)

### use-to-require (default: off)

Consolidate `:require` and `:use` blocks inside ns declarations, rewriting them
using `:require` if possible.

### remove-unused-requires

Use simple heuristics to remove some probably-unused :require statements:

    [foo :as x] ; if there is no x/y in the ns, this is removed
    [foo :refer [x]] ; if x does not appear in the ns, this is removed

## Cljfmt configuration

You can optionally use a config file at `$HOME/.cljfmt` (override with `-c`).
This is a Clojure file containing a single map of options. Here's an example:

```
{:indent-overrides [; Compojure
                    ["GET" "POST" "PUT" "PATCH" "DELETE" "context"] :list-body
                    ; Korma
                    ["korma.core/select"
                     "korma.core/insert"
                     "korma.core/update"
                     "korma.core/delete"] :list-body]
 :thread-first-overrides ["-?>" :normal]}
```

The configuration map may use the following keys:

### :indent-overrides

This is used to customize the indentation rules that cljfmt applies to
particular functions and macros. The value is a sequence of pairs; the first
element of each pair is either a string or sequence of strings; the second
element of the pair is the indentation rule to apply to the given names.

The names may be given with or without a qualifying namespace. If there is an
indent-override for `foo`, it will apply to any list form starting with the
symbol `foo` whether it's written as `foo` or `ns/foo`. If the indent-override
is for `my.ns/foo`, then it only takes effect if:

1. the symbol is written as `my.ns/foo`, or
2. if the symbol is written as `a/foo` and there is a require containing
   `[my.ns :as a]`, or
3. the symbol is written as `foo` and there is a require containing
   `[my.ns :refer [foo]]`.

The allowed indentation rules are as follows:

**:normal** is the default for sequences that introduce no indentation.

``` clojure
[1
 2]
```

**:list** is the default for lists. The first item of the subsequent line is
aligned under the second element of the list.

``` clojure
(foobar 123
        456)
```

**:list-body** is for list forms which have bodies. Subsequent lines are
indented by two spaces.

Examples of builtins which use `:list-body` indentation by default are `fn`,
`when`, and most macros beginning with `def`.

``` clojure
(when-not (zero? x)
  (/ 1 x))
```

**:let** is for let-like forms. This is similar to `:list-body`, but
additionally the first parameter is expected to be a let-style binding vector in
which the even-numbered elements are indented by two spaces.

Examples of builtins that use `:let` by default are `binding`, `dotimes`,
`if-let`, `when-some`, and `loop`.

``` clojure
(let [foobar
        (+ x 10)
      baz
        (+ y 20)]
  (* x y))
```

**:for** is for for-like forms. This is similar to `:let`, but
additionally it supports having a `:let` clause with let-like binding vector
formatted accordingly.

The builtins that use `:for` by default are `for` and `doseq`.

``` clojure
(for [x
        (range 5)
      :let [y
              (- 100 x)]
  (* x y))
```

**:letfn** is used for indenting letfn, where the binding vector contains
function bodies that themselves should be indented as `:list-body`.

``` clojure
(letfn [(twice [x]
           (* x 2))
        (six-times [y]
           (* (twice y) 3))]
  (println "Twice 15 =" (twice 15))
  (println "Six times 15 =" (six-times 15)))
```

**:deftype** is used for macros similar to deftype that define
functions/methods that themselves should be indented as `:list-body`.

Examples of builtins that use `:deftype` style by default are `defprotocol`,
`definterface`, `extend-type`, and `reify`.

``` clojure
(defrecord Foo [x y z]
  Xer
  (foobar [this]
    this)
  (baz [this a b c]
    (+ a b c)))
```

**:cond0** is similar to `:list-body` but the even-numbered arguments are
indented by two spaces. By default this is used for `cond`.

``` clojure
(cond
  (> a 10)
    foo
  (> a 5)
    bar)
```

**:cond1** is like `:cond0` but it ignores the first argument when counting
parameters for indentation. By default `:cond1` is used for `case`, `cond->`,
and `cond->>`.

``` clojure
(case x
  "one"
    1
  "two"
    2)
```

**:cond2** is like `:cond0` but it ignores the first two argument when counting
parameters for indentation. By default `:cond2` is used for `condp`.

``` clojure
(condp = value
  1
    "one"
  2
    "two"
```

**:cond4** is like `:cond0` but it ignores the first four argument when counting
parameters for indentation.

### :thread-first-overrides

This uses the same general paired format as `:indent-overrides`.

`:thread-first-overrides` allows specifying additional thread-first macro forms.
The following varieties are allowed:

**:normal** is for typical thread-first macros such as `->` and `some->`. They
take one argument and then all subsequent arguments have a threaded first
parameter.

**:cond->** is for `cond->` style threading, where every other argument is
threaded (starting with the third one).
