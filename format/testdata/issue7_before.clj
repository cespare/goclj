(ns foobar
  "hello there"
  (:require [c :refer :all]
            [a]
            [b :as blah])
  (:import a.b.c.Foo
           x.y.z.Foo
           [a.b.c Bar]
           [x.y.z Bar Baz]))
