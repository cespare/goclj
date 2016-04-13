# goclj

Go tools for working with Clojure code.

The parse ([GoDoc](http://godoc.org/github.com/cespare/goclj/parse)) and format
([GoDoc](http://godoc.org/github.com/cespare/goclj/format)) packages implement
Clojure code parsing and (formatted) printing, respectively.

cljfmt is a command-line tool (inspired by gofmt) that uses format to read and
reformat Clojure code. Because it parses the code, its transformations are
entirely safe (they cannot change semantics). It takes care of normalizing
code formatting according to Clojure conventions, including:

- Applying standard indentation rules
- Removing trailing whitespace
- Normalizing spacing between elements
- Moving dangling close parens to the same line
- Sorting imports and requires

To install or update, use `go get -u github.com/cespare/goclj/cljfmt`. Here is
the output of `cljfmt -h`:

```
usage: cljfmt [flags] [paths...]
Any directories given will be recursively walked. If no paths are provided,
cljfmt reads from standard input.

Flags:
  -c value
        path to config file (default /Users/caleb/.cljfmt)
  -l    print files whose formatting differs from cljfmt's
  -w    write result to (source) file instead of stdout
```

You can optionally use a config file at `$HOME/.cljfmt` (override with `-c`).
This is a Clojure file containing a single map of options. Here's an example:

```
{:indent-overrides (; Compojure
                    ["GET" "POST" "PUT" "PATCH" "DELETE" "context"] :list-body
                    ; Korma
                    ["select" "insert" "update" "delete"] :list-body)}
```

The configuration map may use the following keys:

### :indent-overrides

This is used to customize the indentation rules that cljfmt applies to
particular functions and macros. The value is a sequence of pairs; the first
element of each pair is either a string or sequence of strings; the second
element of the pair is the indentation rule to apply to the given names.

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
`for`, `when`, and most macros beginning with `def`.

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
parameters for indentation. By default `:cond1` is used for `condp`.

``` clojure
(condp = value
  1
    "one"
  2
    "two"
```
