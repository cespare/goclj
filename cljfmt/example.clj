(ns example
    (:require [a :as b]
              [c :refer [d e f]])
    (:import g
             (h.i.j.K)
             [l.m.n O]))

(def blah
     (let [a
           (foobar blah)
           \b @xyz]
          {:foo 1
           :bar 2}))

(defn f
      "this is a fn
  with a multi-line
  description"
      [x y z]
      (+ x
         y
         z))
