(cond
  (> a 3)
    foo
  (> a 10)
    bar
  :else
    baz)

(case x
  "foo"
    foo
  "bar"
    bar
  baz)

(cond-> 1
  true inc
  false
    (* 42)
  (= 2 2)
    (* 3))

(condp = 123
  abc
    foo
  def
    bar)
