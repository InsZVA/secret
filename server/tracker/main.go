package main

import (
	"net/http"
)

func main() {
	if LoadConfig() != nil {
		panic("Config error!")
	}

	port := Config.getString("listenPort", "8888")

	debug(http.ListenAndServe(":" + port, router))
}
