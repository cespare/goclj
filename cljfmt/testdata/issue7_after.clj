(ns foobar
  "hello there"
  (:require [a]
            [b :as blah]
            [c :refer :all])
  (:import [a.b.c Bar]
           [x.y.z Bar Baz]
           a.b.c.Foo
           x.y.z.Foo))
