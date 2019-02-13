(ns a
  (:require [b]
            [c :refer :all]
            [d :as e :refer [h]]
            [foo-bar.x-y]
            [foo-bar2.x-y]
            [k :refer :all]
            [myspec :as u]
            [o :as p]
            [q :as r])
  (:import foo_bar.x_y.Z
           [foo_bar2.x_y A B C]))

(e/x #'p/x)
(h 3)
(b/x ::k1 ::q/k2 ::u/k3)
#::r{:a 0 :b 1}
