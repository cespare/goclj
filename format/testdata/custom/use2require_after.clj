(ns a
  (:require
    ; b
    ; c
    ; f
    ; e
    ; h
    [a :as b]
    [a :as g :refer [d e f h]]
    [a [b :as c]] ; d
    [c :as d :refer :all]
    ; i
    [i]
    [q0 :as q1]
    [x :refer [z
               y]] ; g
    [z :refer :all]
    #{blah} ; a
    ; below0
    ; below1
    )
  (:use
    3))
