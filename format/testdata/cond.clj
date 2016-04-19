; cond-like forms should align with the first arg, if given on the same line,
; or else align like list bodies.
(cond
  (= 1 2)
    3
  (= 4 5)
    6)

(cond (= 1 2)
        3
      (= 4 5
         6))

(-> {}
    (assoc :foo 1
           :bar
             234))
