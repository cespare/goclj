(ns a
  (:require [b]
            [c :refer :all]
            [d :as e]
            [d :as f :refer [g h i]]
            [foo-bar.x-y :as z]
            [foo-bar2.x-y :refer [z]]
            [j :refer [k l]])
  (:use k)
  (:import foo_bar.x_y.Z
           [foo_bar2.x_y A B C])
  (:require [m :as n]
            [o :as p]
            [q :as r]
            [myspec :as u]))

(e/x #'p/x)
(h 3)
(b/x ::k1 ::q/k2 ::u/k3)
#::r{:a 0 :b 1}
