package main

import (
	"net/http"
	"strings"
)

type Router struct {}

var router Router

func (router Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.RequestURI, "/")
	path = path[1:]

	switch path[0] {
	case "input":
		InputHandler(path, w, r)
		return
	case "stream":
		StreamHandler(path, w, r)
		return
	case "master":
		MasterHandler(path, w, r)
		return
	case "client":
		ClientHandler(path, w, r)
		return
	}

	w.WriteHeader(404)
}
