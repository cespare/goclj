(ns foo.bar
  (:require
    ; a
    [a :refer [x y z]] ; blah
    ; z
    [z :as z] ; another
    ; after
    )
  (:import
    (java.io File)
    (java.io DataInputStream)
    (java.util.zip GZIPInputStream)))
