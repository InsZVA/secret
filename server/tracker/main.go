package main

import (
	"net/http"
)

func main() {
	if LoadConfig() != nil {
		panic("Config error!")
	}

	port := Config.getString("listenPort", "8080")
	http.HandleFunc("/time", TimeHandler)
	http.HandleFunc("/input.webm", InputHandler)
	http.HandleFunc("/stream", StreamHandler)

	http.ListenAndServe(":" + port, nil)
}
