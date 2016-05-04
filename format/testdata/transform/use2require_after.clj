(ns a
  (:require i
            [a :as b]
            [a :as g :refer [d e f h]]
            [c :as d :refer :all]
            [x :refer [y]]
            [z :refer :all]
            #{blah})
  (:use 3))
