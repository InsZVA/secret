package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"encoding/binary"
)

const (
	CACHED_BUFFER_DURATION = 10000 // more than client
)

var (
	videobuffer = VideoBuffer{
		onSliceReady: func (vs VideoSlice) {
			buffer := make([]byte, 20)
			binary.BigEndian.PutUint32(buffer,20)
			binary.BigEndian.PutUint64(buffer[4:], vs.createTimeStamp)
			binary.BigEndian.PutUint64(buffer[12:], vs.nextTimeStamp)
			buffer = append(buffer, vs.data...)
			debug("broadcast")
			broadcast <- buffer
		},
	}

	broadcast = make(chan []byte, 10)

	upgrader             = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

type VideoSlice struct {
	createTimeStamp uint64
	nextTimeStamp uint64
	data []uint8
}

type VideoBuffer struct {
	buffer []VideoSlice
	onSliceReady func (VideoSlice)
}

func (vb *VideoBuffer) push(vs VideoSlice) {
	if len(vb.buffer) != 0 {
		// Ensure every slice in buffer is continuous
		vb.buffer[len(vb.buffer) - 1].nextTimeStamp = vs.createTimeStamp
	}
	vb.buffer = append(vb.buffer, vs)
	if vs.createTimeStamp - vb.buffer[0].createTimeStamp > CACHED_BUFFER_DURATION {
		if vb.onSliceReady != nil {
			vb.onSliceReady(vb.buffer[0])
		}
		vb.buffer = vb.buffer[1:]
	}
}

func StreamHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	for {
		select {
		case buffer := <-broadcast:
			if nil != conn.WriteMessage(websocket.BinaryMessage, buffer) {
				return
			}
		}
	}
}