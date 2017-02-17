package main

import (
	"net/http"
	"strings"
	"sync"
	"io/ioutil"
	"strconv"
)

var streamMap = StreamMap {m: make(map[string]*Stream)}

type StreamMap struct {
	m map[string]*Stream
	lock sync.RWMutex
}

func (sm *StreamMap) Set(k string, v *Stream) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.m[k] = v
}

func (sm *StreamMap) Get(k string) *Stream {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	return sm.m[k]
}

func (sm *StreamMap) Remove(k string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	delete(sm.m, k)
}

func InputHandler(path []string, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(403); return
	}
	if len(path) < 3 {
		w.WriteHeader(403); return
	}

	sid := path[1]
	fname := path[2]

	// {v|a}_{codec}.hdr
	// {v|a}_{codec}_%d.chk
	fts := strings.Split(fname, "_")
	points := strings.Split(fts[len(fts) - 1], ".")
	var err error
	defer r.Body.Close()

	switch len(fts) {
	case 2:
		if points[1] != "hdr" {
			w.WriteHeader(403); return
		}
		stream := streamMap.Get(sid)
		if stream == nil {
			stream = (&Stream{}).init()
			streamMap.Set(sid, stream)
		}

		if fts[0] == "v" {
			stream.track[0].codec = points[0]
			stream.track[0].initChunk, err = ioutil.ReadAll(r.Body)
			debug("Video " + sid + " header:" + points[0])
			if err != nil {
				w.WriteHeader(403); return
			}
			if !stream.sw.Inited() {
				stream.sw.Reset()
			}
			w.WriteHeader(200); return
		}
		if fts[0] == "a" {
			stream.track[1].codec = points[0]
			stream.track[1].initChunk, err = ioutil.ReadAll(r.Body)
			debug("Audio " + sid + " header:" + points[0])
			if err != nil {
				w.WriteHeader(403); return
			}
			if !stream.sw.Inited() {
				stream.sw.Reset()
			}
			w.WriteHeader(200); return
		}
		w.WriteHeader(403); return
	case 3:
		if points[1] != "chk" {
			w.WriteHeader(403); return
		}
		stream := streamMap.Get(sid)
		if stream == nil {
			w.WriteHeader(404); return
		}

		//buff := make([]byte, 512*1024)
		buff, err := ioutil.ReadAll(r.Body)
		//n, err := r.Body.Read(buff)
		//buff = buff[:n]
		if err != nil {
			w.WriteHeader(403); return
		}
		id, err := strconv.Atoi(points[0])
		if err != nil {
			w.WriteHeader(403); return
		}
		ck := Chunk{
			id: uint32(id),
			data: buff,
		}
		if fts[0] == "v" {
			stream.sw.AWait()
			ck.codec = stream.track[0].codec
			stream.track[0].buffer.push(ck)
			debug("Video " + sid + " buffered")
			stream.sw.ARelease()
			w.WriteHeader(200); return
		}
		if fts[0] == "a" {
			stream.sw.BWait()
			ck.codec = stream.track[1].codec
			stream.track[1].buffer.push(ck)
			debug("Audio " + sid + " buffered")
			stream.sw.BRelease()
			w.WriteHeader(200); return
		}
		w.WriteHeader(403); return
	}
	w.WriteHeader(403); return
}