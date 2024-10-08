package queue

import (
	"testing"
)

func TestPushPop(t *testing.T) {
	queue := NewRAQueue()
	data := []string{"first", "second", "third"}
	for _, str := range data {
		queue.Push(str)
	}

	for _, str := range data {
		if queue.Pop().(string) != str {
			t.Fail()
		}
	}

	if queue.Pop() != nil {
		t.Fail()
	}
}

func TestDelete(t *testing.T) {
	queue := NewRAQueue()
	data := []string{"first", "second", "third"}
	idx := make([]uint, 3)
	for i, str := range data {
		idx[i] = queue.Push(str)
	}

	if queue.Delete(idx[1]).(string) != "second" {
		t.Fail()
	}

	data = []string{"first", "third"}
	for _, str := range data {
		if queue.Pop().(string) != str {
			t.Fail()
		}
	}

	if queue.Pop() != nil {
		t.Fail()
	}
}
