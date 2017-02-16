package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"encoding/binary"
	"container/list"
	"sync"
)

var (
	CACHED_LENGTH = 2 // same as client
	TRANSPORT = "chunk"
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

func (t *Track) encodeInitMsg() []byte {
	buff := make([]byte, 4)
	buff = append(buff, []byte(t.codec)...)
	binary.BigEndian.PutUint32(buff, uint32(len(buff)))
	buff = append(buff, t.initChunk...)
	return buff
}

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

type Chunk struct {
	id uint32
	codec string
	data []uint8
}

func (cs *Chunk) encode() []byte {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint32(buffer[4:], cs.id)
	buffer = append(buffer, []byte(cs.codec)...)
	binary.BigEndian.PutUint32(buffer, uint32(len(buffer)))
	buffer = append(buffer, cs.data...)
	return buffer
}

func (cs *Chunk) split(n int) [][]byte {
	ret := make([][]byte, n)
	sliceSize := (len(cs.data) + n - 1) / n
	for i := 0; i < n - 1; i++ {
		ret[i] = Slice{
			cs.id,
			uint32(i),
			uint32(n),
			cs.codec,
			cs.data[i*sliceSize:(i+1)*sliceSize],
		}.encode()
	}
	ret[n-1] = Slice{
		cs.id,
		uint32(n-1),
		uint32(n),
		cs.codec,
		cs.data[(n-1)*sliceSize:],
	}.encode()
	return ret
}

type Slice struct {
	cid uint32
	sid uint32
	stotal uint32
	codec string
	data []uint8
}

func (s Slice) encode() []byte {
	ret := make([]byte, 16)
	binary.BigEndian.PutUint32(ret[4:], s.cid)
	binary.BigEndian.PutUint32(ret[8:], s.sid)
	binary.BigEndian.PutUint32(ret[12:], s.stotal)
	ret = append(ret, []byte(s.codec)...)
	binary.BigEndian.PutUint32(ret, uint32(len(ret)))
	ret = append(ret, s.data...)
	return ret
}

type ChunkBuffer struct {
	buffer []Chunk
	stream *Stream
	lock sync.RWMutex
}

func (cb *ChunkBuffer) onSliceRemove(cs Chunk) {
	//debug("broadcast")
}

var ChunkTranport = func(conn *Node, cs Chunk) {
	select {
	case conn.ch <- cs.encode():
	default:
	}
}

// TODO: fairly transport
var SliceTransport = func(conn *Node, cs Chunk) {
	buffer := cs.split(4)
	for i := 0; i < 4; i++ {
		select {
		case conn.ch <- buffer[i]:
		default:
		}
	}
}

func (cb *ChunkBuffer) push(cs Chunk) {
	cb.lock.Lock()
	defer cb.lock.Unlock()

	cb.buffer = append(cb.buffer, cs)
	if len(cb.buffer) > CACHED_LENGTH {
		cb.onSliceRemove(cb.buffer[0])
		cb.buffer = cb.buffer[1:]
	}

	cb.stream.iterateConn(func (conn *Node) {
		if TRANSPORT == "chunk" {
			ChunkTranport(conn, cs)
		} else {
			SliceTransport(conn, cs)
		}
	})
}

func (cb *ChunkBuffer) fastload(conn *Node) {
	cb.lock.RLock() // TODO: lock-free
	defer cb.lock.RUnlock()
	for i := 0; i < len(cb.buffer); i++ {
		cs := cb.buffer[i]
		conn.ch <- cs.encode()
	}
}

func (s *Stream) fastload(conn *Node) {
	s.track[0].buffer.lock.RLock()
	s.track[1].buffer.lock.RLock()
	defer s.track[0].buffer.lock.RUnlock()
	defer s.track[1].buffer.lock.RUnlock()
	min := len(s.track[0].buffer.buffer)
	max := &s.track[1]
	if len(s.track[1].buffer.buffer) < min {
		min = len(s.track[1].buffer.buffer)
		max = &s.track[0]
	}
	for i := 0; i < min; i++ {
		cs := s.track[0].buffer.buffer[i]
		conn.ch <- cs.encode()
		cs = s.track[1].buffer.buffer[i]
		conn.ch <- cs.encode()
	}
	for i := min; i < len(max.buffer.buffer); i++ {
		cs := max.buffer.buffer[i]
		conn.ch <- cs.encode()
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

	conn.WriteMessage(websocket.BinaryMessage, stream.track[0].encodeInitMsg())
	conn.WriteMessage(websocket.BinaryMessage, stream.track[1].encodeInitMsg())

	//fastload
	stream.fastload(node)

	stream.hangConn(node)
	defer stream.releaseConn(node)

	for {
		buffer := <-node.ch
		if nil != conn.WriteMessage(websocket.BinaryMessage, buffer) {
		}
	}
}