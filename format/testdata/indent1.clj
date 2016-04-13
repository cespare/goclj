(locking x
  (dotimes [n 1000]
    (with-redefs [y z]
      (send-foo 1
        (with-bar 2
          (delete 3
                  (up 4
                      blah)))))))
