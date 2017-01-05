(cond-> {}
  true (assoc :a "a"
              :b "b"))

(cond->
  {}
  true (assoc :a "a"
              :b "b"))

(cond->
  {}
  true
    (assoc :a "a"
           :b "b"))

(-> (assoc {}
      :a
        "a"
      :b
        "b")
    (assoc
      :c
        "c"
      :d
        "d"))
