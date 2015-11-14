; https://github.com/bbatsov/clojure-style-guide

(when something
  (something-else))

(with-out-str
  (println "Hello, ")
  (println "world!"))

(filter even?
        (range 1 10))

; TODO: reverse decision from #9
;(filter
;  even?
;  (range 1 10))
;
;(or
;  ala
;  bala
;  portokala)

(let [thing1 "some stuff"
      thing2 "other stuff"]
  {:thing1 thing1
   :thing2 thing2})

; TODO: move arg vector to same line
;(defn foo
;  [x] (bar x))

; TODO: move the dispatch-val to the same line
;(defmethod foo
;  :bar
;  [x]
;  (baz x))

(defn foo
  "I have two arities."
  ([x]
   (foo x 1))
  ([x y]
   (+ x y)))

; TODO: Fix #6
;(defn foo
;  "Hello there. This is
;a multi-line docstring."
;  []
;  (bar))

(foo (bar baz) quux)
(foo (bar baz) quux)

[1 2 3]
(1 2 3)

(when something
  (something-else))
