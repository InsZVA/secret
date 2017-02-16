package main

import (
	"net/http"
	"strconv"
)

// Http debug helper
func ClientHandler(path []string, w http.ResponseWriter, r *http.Request) {
	if len(path) == 1 {
		clientMap.lock.RLock()
		defer clientMap.lock.RUnlock()
		html := "<html><body><ul>"
		html += "<li><a href='/client/server'>server</a></li>"
		for h, c := range clientMap.m {
			html += "<li><a href='/client/" + h + "'>" + h + " Level:" +
				strconv.Itoa(c.Level()) + "</a></li>"
		}
		html += "</ul><p><a href='/client'>Client List</a></p></body></html>"
		w.Write([]byte(html))
		return
	}

	if len(path) >= 2 {
		id := path[1]
		cli := clientMap.Get(id)
		if id == "server" {
			cli = &server
		}
		if cli == nil {
			w.WriteHeader(404)
			return
		}
		html := "<html><body>"
		html += cli.InfoHTML()
		html += "<p><a href='/client'>Client List</a></p></body></html>"
		w.Write([]byte(html))
	}
}
