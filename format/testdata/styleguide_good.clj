; https://github.com/bbatsov/clojure-style-guide

(when something
  (something-else))

(with-out-str
  (println "Hello, ")
  (println "world!"))

(filter even?
        (range 1 10))

; TODO: see #27
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

(defn foo [x]
  (bar x))

(defmethod foo :bar
  [x]
  (baz x))

(defmethod foo :bar
  [x]
  (baz x))

(defn foo
  "I have two arities."
  ([x]
   (foo x 1))
  ([x y]
   (+ x y)))

(defn foo
  "Hello there. This is
  a multi-line docstring."
  []
  (bar))

(foo (bar baz) quux)
(foo (bar baz) quux)

[1 2 3]
(1 2 3)

(when something
  (something-else))
