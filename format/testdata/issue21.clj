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

(condp = 123
  abc
    foo
  def
    bar)

