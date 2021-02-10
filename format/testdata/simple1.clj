(ns foo.bar
  (:require
    [clojure.set :as set]
    [clojure.string :as string]))

(defn throw-str
  "Throws a RuntimeException, str-ing together its arguments."
  [& args]
  (throw
    (RuntimeException. ^String (apply str
                                      args))))

(defn str-starts-with?
  "Returns whether s begins with prefix."
  [^String s prefix]
  (.startsWith s
               prefix))
