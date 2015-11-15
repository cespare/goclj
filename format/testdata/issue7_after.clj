(ns foobar
  "hello there"
  (:require [a]
            [b :as blah]
            [c :refer :all])
  (:import a.b.c.Foo
           x.y.z.Foo
           [a.b.c Bar]
           [x.y.z Bar Baz]))
