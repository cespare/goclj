(some-> {}
        (assoc
          :foo
            123
          :bar
            456)
        (-> (case
              "1"
                (foo 1)
              "2"
                (foo 2)
              :default)))
