package main

import (
    "github.com/petar/GoLLRB/llrb"
)

const MaxUint = ^uint(0)
const WrapTreshold = (MaxUint >> 1) + 1

type RAQueue struct {
    llrb *llrb.LLRB
    cur_lsn uint
}

type queueElem struct {
    lsn uint
    payload interface{}
}

func (e *queueElem) Less (than llrb.Item) bool {
    x, y := e.lsn, than.(*queueElem).lsn
    return (x < y && (y - x) <= WrapTreshold) || (x > y && (x - y) > WrapTreshold)
}

func NewRAQueue() *RAQueue {
    return &RAQueue{llrb: llrb.New()}
}

func (q *RAQueue) Push(e interface{}) uint {
    lsn := q.cur_lsn
    q.cur_lsn++
    elem := &queueElem{lsn: lsn, payload: e}
    q.llrb.ReplaceOrInsert(elem)
    return lsn
}

func (q *RAQueue) Pop() interface{} {
    min := q.llrb.DeleteMin()
    if min == nil {
        return nil
    } else {
        return min.(*queueElem).payload
    }
}

func (q *RAQueue) Delete(key uint) interface{} {
    elem := q.llrb.Delete(&queueElem{lsn: key})
    if elem == nil {
        return nil
    } else {
        return elem.(*queueElem).payload
    }
}
