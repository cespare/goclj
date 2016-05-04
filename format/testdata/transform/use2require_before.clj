(ns a
  (:require #{blah} ; a
            ; b
            ; c
            a
            [a [b :as c]] ; d
            [a :as b] ; e
            (c :as d)
            ; f
            [a :refer [f e
                       d]]
            ; below0
            )
  (:use c
        (x :only [z
                  y]) ; g
        [z]
        3
        ; below1
        )
  (:require [a :as g :refer [h]] ; h
            ; i
            i))
