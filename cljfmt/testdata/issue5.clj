(let [foo 123
      bar
        (baz asdf)]
  nil)

(let [foo ; bar
        baz]
  nil)

(let [^String foo
        "asdf"]
  nil)
