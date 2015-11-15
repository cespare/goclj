(ns foo.bar
  (:require ; a
            [a :refer [x y z]] ; blah
            ; z
            [z :as z] ; another
            ; after
            )
  (:import java.io.DataInputStream
           java.io.File
           [java.util.zip GZIPInputStream]))
