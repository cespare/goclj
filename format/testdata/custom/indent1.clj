(ns myns
  (:require
    [org.lib1
     :as lib2]
    [org.lib3 :refer [mycond] :refer-macros [mymacro1]])
  (:require-macros
    [org.lib4 :refer [mymacro2]]))

(locking x
  (dotimes [n 1000]
    (with-redefs [y z]
      (send-off 1
        (with-bar 2
          (delete 3
                  (x/up 4
                        (org.lib1/f 5
                                    (lib2/f 6
                                            (mycond
                                              (> z 3)
                                              "three"
                                              (> z 7)
                                              (f 7
                                                 8
                                                 9))
                                            (mymacro1 [a 10]
                                                      "macro")
                                            (mymacro2 [a 10]
                                                      "macro"))))))))))
