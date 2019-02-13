#?(:clj 1
   :default 2)
(println #?@(:cljs
               "xyz"
             :clj
               "blah"))
