package main

import (
	"github.com/huandu/skiplist"
)

const MaxUint = ^uint(0)
const WrapTreshold = (MaxUint >> 1) + 1

type RAQueue struct {
	l       *skiplist.SkipList
	cur_lsn uint
}

func NewRAQueue() *RAQueue {
	return &RAQueue{
		l: skiplist.New(skiplist.GreaterThanFunc(func(lhs, rhs interface{}) int {
			x, y := lhs.(uint), rhs.(uint)
			switch {
			case x == y:
				return 0
			case (x < y && (y-x) <= WrapTreshold) || (x > y && (x-y) > WrapTreshold):
				return -1
			default:
				return 1
			}
		})),
	}
}

func (q *RAQueue) Push(e interface{}) uint {
	lsn := q.cur_lsn
	q.cur_lsn++
	q.l.Set(lsn, e)
	return lsn
}

func (q *RAQueue) Pop() interface{} {
	if q.l.Len() == 0 {
		return nil
	}
	return q.l.RemoveFront().Value
}

func (q *RAQueue) Delete(key uint) interface{} {
	elem := q.l.Remove(key)
	if elem == nil {
		return nil
	} else {
		return elem.Value
	}
}
