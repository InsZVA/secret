package main

import (
	"net/http"
	"time"
	"strconv"
)

func InputHandler(w http.ResponseWriter, r *http.Request) {
	/*
	width, err := strconv.Atoi(r.URL.Query().Get("width"))
	if err != nil {
		w.WriteHeader(403); return
	}
	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil {
		w.WriteHeader(403); return
	}
	mime := r.URL.Query().Get("mime")
	*/
	if r.Method != "POST" {
		w.WriteHeader(403); return
	}

	const CHUNK_SIZE = 100 * 1024
	for {
		buffer := make([]byte, CHUNK_SIZE)
		n, e := r.Body.Read(buffer)
		if e != nil {
			return
		}
		debug("read buffer:" + strconv.Itoa(n))
		videobuffer.push(VideoSlice{
			createTimeStamp: uint64(time.Now().UnixNano() / 1000000),
			data: buffer[:n],
		})
	}
}