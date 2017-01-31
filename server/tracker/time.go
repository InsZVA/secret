package main

import (
	"net/http"
	"time"
	"strconv"
)

func TimeHandler(w http.ResponseWriter, r *http.Request) {
	timestamp := time.Now().UnixNano() / 1000000
	w.Write([]byte(strconv.Itoa(int(timestamp))))
}
