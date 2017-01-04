(ns foobar
  "hello there"
  (:require [a]
            [b :as blah]
            [c :refer :all])
  (:import [a.b.c Bar]
           a.b.c.Foo
           [x.y.z Bar Baz]
           x.y.z.Foo))
