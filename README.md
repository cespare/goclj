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
{:indent-special ["GET" "POST" "PUT" "PATCH" "DELETE" "context" ; Compojure
                  "select" "insert" "update" "delete" ; Korma
                  ]}
```

The configuration map may use the following keys:

**:indent-special** is used to indicate a set of custom names which should cause
inner expressions on following lines to be indented by 2 spaces rather than
being aligned after the end of the name. For many well-known functions and
macros, cljfmt already uses a special indent:

``` clojure
(when-not x
  blah)
```

but for other names, subsequent lines are aligned like this:

``` clojure
(foo-bar x
         blah)
```

If you add `"foo-bar"` to the `:indent-special` list, it would be indented like
this instead:

``` clojure
(foo-bar x
  blah)
```
