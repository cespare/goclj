(ns a
  (:require i
            [a :as b]
            [a :refer [d e f h] :as g]
            [c :refer :all :as d]
            [x :refer [y]]
            [z :refer :all]
            #{blah})
  (:use 3))
