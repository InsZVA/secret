package main

import (
	"net/http"
)

func main() {
	if LoadConfig() != nil {
		panic("Config error!")
	}

	port := Config.getString("listenPort", "8888")
	CACHED_LENGTH = Config.getInt("cacheLength", 0)
	TRANSPORT = Config.getString("transport", "chunk")

	debug(http.ListenAndServe(":" + port, router))
}
