package main

import (
	"sync/atomic"
)

/**
	a lock-free link-list based set
	inspired by http://blog.csdn.net/iter_zc/article/details/41115021
 */
type lfset struct {
	head *lfnode
}

type lfnode struct {
	key uint32
	value hasher
	next atomic.Value
}

type markedpointer struct {
	mark bool
	node *lfnode
}

type hasher interface {
	hash() uint32
}

func NewLFSet() *lfset {
	ret := &lfset{
		head: &lfnode{
			key: 0,
		},
	}
	ret.head.next.Store(markedpointer{
		mark: false,
		node: &lfnode{
			key: 0xffffffff,
		},
	})
	return ret
}

func (set *lfset) add(v hasher) bool {
	key := v.hash()
	prev := set.head
	curr := prev.next.Load().(*lfnode)
	for curr.key < key {
		if curr.next.Load() == nil {
			break
		}
		prev = curr
		curr = prev.next.Load().(markedpointer).node
	}
	if curr.key == key {
		return false
	}

	node := &lfnode{
		key: key,
		value: v,
	}
	node.next.Store(markedpointer{
		mark: false,
		node: curr,
	})
	//TODO
	return true
}