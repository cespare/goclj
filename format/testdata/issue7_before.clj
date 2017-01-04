(ns foobar
  "hello there"
  (:require [c :refer :all]
            [a]
            [b :as blah])
  (:import [a.b.c Bar]
           a.b.c.Foo
           x.y.z.Foo
           [x.y.z Bar Baz]))
