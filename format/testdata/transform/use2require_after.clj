(ns a
  (:require [a :as b]
            [a :as g :refer [d e f h]]
            [a [b :as c]]
            [c :as d :refer :all]
            [i]
            [x :refer [z
                       y]]
            [z :refer :all]
            #{blah})
  (:use 3))
