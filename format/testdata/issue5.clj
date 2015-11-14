(let [foo 123
      bar
        (baz asdf)
      quux 234]
  nil)

(let [foo ; bar
        baz]
  nil)

(let [^String foo
        "asdf"]
  nil)
