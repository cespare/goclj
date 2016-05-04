# goclj

> Go tools for working with Clojure code.

The parse ([GoDoc](http://godoc.org/github.com/cespare/goclj/parse)) and format
([GoDoc](http://godoc.org/github.com/cespare/goclj/format)) packages implement
Clojure code parsing and (formatted) printing, respectively.

cljfmt is a command-line tool (inspired by gofmt) that uses format to read and
reformat Clojure code. Because it parses the code, its formatting
transformations are safe (they cannot change semantics).

Additionally, cljfmt applies various transformations to the code; these are
discussed in the **Transforms** section, below.

To install or update, use `go get -u github.com/cespare/goclj/cljfmt`. Here is
the output of `cljfmt -h`:

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

### use-to-require (default: off)

Consolidate `:require` and `:use` blocks inside ns declarations, rewriting them
using `:require` if possible.

## Cljfmt configuration

You can optionally use a config file at `$HOME/.cljfmt` (override with `-c`).
This is a Clojure file containing a single map of options. Here's an example:

```
{:indent-overrides [; Compojure
                    ["GET" "POST" "PUT" "PATCH" "DELETE" "context"] :list-body
                    ; Korma
                    ["select" "insert" "update" "delete"] :list-body]
 :thread-first-overrides ["-?>" :normal]}
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

### :thread-first-overrides

This uses the same general paired format as `:indent-overrides`.

`:thread-first-overrides` allows specifying additional thread-first macro forms.
The following varieties are allowed:

**:normal** is for typical thread-first macros such as `->` and `some->`. They
take one argument and then all subsequent arguments have a threaded first
parameter.

**:cond->** is for `cond->` style threading, where every other argument is
threaded (starting with the third one).
