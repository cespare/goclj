(ns a
  (:require [b]
            [c :refer :all]
            [d :as e]
            [d :as f :refer [g h i]]
            [j :refer [k l]])
  (:use k)
  (:require [m :as n]
            [o :as p]))

(e/x #'p/x)
(h 3)
