(ns foo.bar
  (:require
    ; z
    [z :as z] ; another
    ; a
    [a :refer [x y z]] ; blah
    ; after
    )
  (:import
    java.io.File
    (java.util.zip GZIPInputStream)
    java.io.DataInputStream
    ))
