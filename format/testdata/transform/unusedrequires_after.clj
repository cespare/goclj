(ns a
  (:require [b]
            [c :refer :all]
            [d :as e :refer [h]]
            [k :refer :all]
            [o :as p]))

(e/x #'p/x)
(h 3)
