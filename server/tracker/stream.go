package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"encoding/binary"
	"container/list"
	"sync"
)

const (
	CACHED_BUFFER_DURATION = 5500 // more than client
)

// 0: Video
// 1: Audio
type Stream struct {
	track [2]Track

	chanlist list.List  // TODO: minimum lock
	lock     sync.RWMutex // TODO: lock list
}

func (s *Stream) init() *Stream {
	s.chanlist.Init()
	s.track[0].buffer.stream = s
	s.track[1].buffer.stream = s
	return s
}

func (s *Stream) hangConn(conn *Node) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.chanlist.PushBack(conn)
}

func (s *Stream) releaseConn(conn *Node) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for l := s.chanlist.Front(); l != nil; l = l.Next() {
		if l.Value.(*Node) == conn {
			l.Value.(*Node).Close()
			s.chanlist.Remove(l)
			return
		}
	}
}

func (s *Stream) iterateConn(handler func (conn *Node)) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for l := s.chanlist.Front(); l != nil; l = l.Next() {
		handler(l.Value.(*Node))
	}
}

type Track struct {
	initChunk []byte
	codec string
	buffer ChunkBuffer
}

var (
	broadcast = make(chan []byte, 10)

	upgrader             = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

type ChunkSlice struct {
	createTimeStamp uint64
	nextTimeStamp uint64
	codec string
	data []uint8
}

type ChunkBuffer struct {
	buffer []ChunkSlice
	stream *Stream
	lock sync.RWMutex
}

func (cb *ChunkBuffer) onSliceRemove(cs ChunkSlice) {
	//debug("broadcast")

}

func (cb *ChunkBuffer) push(cs ChunkSlice) {
	cb.lock.Lock()
	defer cb.lock.Unlock()

	if len(cb.buffer) != 0 {
		// Ensure every slice in buffer is continuous
		cb.buffer[len(cb.buffer) - 1].nextTimeStamp = cs.createTimeStamp
	}
	cb.buffer = append(cb.buffer, cs)
	if cs.createTimeStamp - cb.buffer[0].createTimeStamp > CACHED_BUFFER_DURATION {
		cb.onSliceRemove(cb.buffer[0])
		cb.buffer = cb.buffer[1:]
	}

	if len(cb.buffer) > 1 {
		cs = cb.buffer[len(cb.buffer) - 2]
		buffer := make([]byte, 20)
		binary.BigEndian.PutUint64(buffer[4:], cs.createTimeStamp)
		binary.BigEndian.PutUint64(buffer[12:], cs.nextTimeStamp)
		buffer = append(buffer, []byte(cs.codec)...)
		binary.BigEndian.PutUint32(buffer, uint32(len(buffer)))
		buffer = append(buffer, cs.data...)

		cb.stream.iterateConn(func(conn *Node) {
			select {
			case conn.ch <- buffer:
			default:
			}
		})
	}
}

func (cb *ChunkBuffer) fastload(conn *Node) {
	cb.lock.RLock() // TODO: lock-free
	defer cb.lock.RUnlock()
	for i := 0; i < len(cb.buffer) - 1; i++ {
		cs := cb.buffer[i]
		buffer := make([]byte, 20)
		binary.BigEndian.PutUint64(buffer[4:], cs.createTimeStamp)
		binary.BigEndian.PutUint64(buffer[12:], cs.nextTimeStamp)
		buffer = append(buffer, []byte(cs.codec)...)
		binary.BigEndian.PutUint32(buffer,uint32(len(buffer)))
		buffer = append(buffer, cs.data...)
		conn.ch <- buffer
	}
}

func StreamHandler(path []string, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(500); return
	}
	if len(path) < 2 {
		w.WriteHeader(403); return
	}
	sid := path[1]
	stream := streamMap.Get(sid)
	if stream == nil {
		w.WriteHeader(404); return
	}

	node := &Node{
		inport: conn,
		ch:     make(chan []byte, 10),
	}

	buff := make([]byte, 4)
	buff = append(buff, []byte(stream.track[0].codec)...)
	binary.BigEndian.PutUint32(buff, uint32(len(buff)))
	buff = append(buff, stream.track[0].initChunk...)
	conn.WriteMessage(websocket.BinaryMessage, buff)

	buff = make([]byte, 4)
	buff = append(buff, []byte(stream.track[1].codec)...)
	binary.BigEndian.PutUint32(buff, uint32(len(buff)))
	buff = append(buff, stream.track[1].initChunk...)
	conn.WriteMessage(websocket.BinaryMessage, buff)

	//fastload
	go stream.track[1].buffer.fastload(node)
	stream.track[0].buffer.fastload(node)

	stream.hangConn(node)
	defer stream.releaseConn(node)

	for {
		buffer := <-node.ch
		if nil != conn.WriteMessage(websocket.BinaryMessage, buffer) {
		}
	}
}