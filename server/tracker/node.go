package main

import "github.com/gorilla/websocket"

type Node struct {
	conn *websocket.Conn
	inport *websocket.Conn
	ch chan []byte
}

func (n *Node) Close() {
	n.inport.Close()
}