(ns a
  (:require #{blah}
            a
            [a :as b]
            (c :as d)
            [a :refer [f e
                       d]])
  (:use c
        (x :only [z
                  y])
        [z]
        3)
  (:require [a :as g :refer [h]]
            i))
