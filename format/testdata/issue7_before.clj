(ns foobar
  "hello there"
  (:require [c :refer :all]
            [a]
            [b :as blah])
  (:import x.y.z.Foo
           [x.y.z Bar Baz]
           a.b.c.Foo
           [a.b.c Bar]))
