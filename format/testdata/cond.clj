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

(condp some x #{0 1} "0 or 1" #{2 3} "two or three" #{4 5} :>> #(str "found " % \"))
(condp some x #{0 1} "0 or 1" #{2 3} "two or three" #{4 5} :>>
    #(str "found " % \"))
(condp some x #{0 1} "0 or 1" #{2 3} "two or three" #{4 5}
    :>>
    #(str "found " % \"))
(condp some x #{0 1} "0 or 1" #{2 3} "two or three"
  #{4 5}
    :>>
    #(str "found " % \"))
(condp some x #{0 1} "0 or 1" #{2 3}
    "two or three"
  #{4 5}
    :>>
    #(str "found " % \"))
(condp some x #{0 1} "0 or 1"
  #{2 3}
    "two or three"
  #{4 5}
    :>>
    #(str "found " % \"))
(condp some x #{0 1}
    "0 or 1"
  #{2 3}
    "two or three"
  #{4 5}
    :>>
    #(str "found " % \"))
(condp some x
  #{0 1}
    "0 or 1"
  #{2 3}
    "two or three"
  #{4 5}
    :>>
    #(str "found " % \"))
(condp some
  x
  #{0 1}
    "0 or 1"
  #{2 3}
    "two or three"
  #{4 5}
    :>>
    #(str "found " % \"))
(condp
  some
  x
  #{0 1}
    "0 or 1"
  #{2 3}
    "two or three"
  #{4 5}
    :>>
    #(str "found " % \"))
