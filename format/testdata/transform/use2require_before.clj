(ns a
  (:require #{blah}
            a
            [a :as b]
            (c :as d)
            [a :refer [d e f]])
  (:use c
        (x :only [y])
        [z]
        3)
  (:require [a :as g :refer [h]]
            i))
