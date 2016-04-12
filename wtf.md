# Clojure parsing WTFs

No spec of any kind :( The closest is [the reader documentation](http://clojure.org/reader),
but this is sparse, incomplete, and uses imprecise language.

## Lexing/parsing edge cases

* Reader doc says "Symbols begin with a non-numeric character and can contain
  alphanumeric characters and `*`, `+`, `!`, `-`, `_`, and `?` (other characters
  will be allowed eventually, but not all macro characters have been
  determined)"
  - So `+foo` is a valid symbol. However, `+3foo` is interpreted as an (invalid)
    number.
* Doc says "Numbers - generally represented as per Java". However, there are
  several kinds of numeric literals that Java supports which Clojure does not
  recognize; this is undocumented:
  - Binary literals (e.g. `0b1101`) are unrecognized, even though hex literals (`0x42`) are
  - Long, float, and double suffixes (e.g. `1.234f`)
  - Java 7+ allows for underscores in numbers for readability: `123_456`
* There are several undocumnented dispatch macro forms: `#^`, `#!`, `#=`, `#<`.
