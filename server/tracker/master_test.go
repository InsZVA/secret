package main

import (
	"testing"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type Browser struct {
	conn *websocket.Conn
}

func (b *Browser) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8888/master", nil)
	b.conn = conn
	return err
}

func (b *Browser) GetId() string {
	m := make(map[string]interface{})
	m["type"] = "getid"
	b.conn.WriteJSON(m)
	b.conn.ReadJSON(&m)
	return m["id"].(string)
}

func (b *Browser) Find() []string {
	m := make(map[string]interface{})
	m["type"] = "find"
	b.conn.WriteJSON(m)
	for {
		b.conn.ReadJSON(&m)
		if m["type"].(string) == "peek" {
			l := len(m["peek"].([]interface{}))
			ret := make([]string, l)
			for i := 0; i < l; i++ {
				ret[i] = m["peek"].([]interface{})[i].(string)
			}
			return ret
		}
	}
	return nil
}

func (b *Browser) Bind(id string, reserved bool) {
	m := make(map[string]interface{})
	m["type"] = "bind"
	m["id"] = id
	m["reserved"] = reserved
	b.conn.WriteJSON(m)
}

func (b *Browser) Evaluate(id string, value float64) {
	m := make(map[string]interface{})
	m["type"] = "evaluate"
	m["id"] = id
	m["value"] = value
	b.conn.WriteJSON(m)
}

func init() {
	go main()
}

func TestClient_translateMessage(t *testing.T) {
	//Client1
	//getid
	b := &Browser{}
	if e := b.Connect(); e != nil {
		t.Error(e)
	}
	log.Println("GetID:")
	log.Println(b.GetId())
	// find
	log.Println("Find:")
	peeked := b.Find() // [Server]
	log.Println(peeked)
	// bind
	b.Bind(peeked[0], false)
	log.Println("Bind")

	//Client2
	//getid
	b2 := &Browser{}
	if e := b2.Connect(); e != nil {
		t.Error(e)
	}
	log.Println("GetID:")
	log.Println(b2.GetId())
	// find
	log.Println("Find:")
	peeked = b2.Find() // [Server]
	log.Println(peeked)
	// bind
	b2.Bind(peeked[0], false)
	log.Println("Bind")

	//Client3
	//getid
	b3 := &Browser{}
	if e := b3.Connect(); e != nil {
		t.Error(e)
	}
	log.Println("GetID:")
	log.Println(b3.GetId())
	// find
	log.Println("Find:")
	peeked = b3.Find() // [Client1, Client2]
	log.Println(peeked)
	// bind Client1
	b3.Bind(peeked[0], false)
	log.Println("Bind:" + peeked[0])
	// evaluate Client2
	b3.Evaluate(peeked[1], 0)
	log.Println("Evaluate:0:" + peeked[1])

	time.Sleep(time.Second)
	log.Println("000")
}

//TODO:
/*
Client3:
JSON	{type: "getid"}
	TEXT	true	true	0:3:18.335
	48	15 B
JSON	{type: "find"}
	TEXT	true	true	0:3:18.335
	48	54 B
JSON	{id: "0cdd78b4c007db9a3d5d1700b0111da5", type: "id"}
	TEXT	false	true	0:3:18.335
	48	91 B
JSON	{cmd: "start", dst: "f9f770ac91089346c1b0c0042ca4da1e", msg: "bind", more...}
cmd	"start"
dst	"f9f770ac91089346c1b0c0042ca4da1e"
msg	"bind"
type	"transaction"
	TEXT	false	true	0:3:18.335
	48	91 B
JSON	{cmd: "start", dst: "01d120f388c983df2dcbcd4b9c27dd26", msg: "bind", more...}
cmd	"start"
dst	"01d120f388c983df2dcbcd4b9c27dd26"
msg	"bind"
type	"transaction"
	TEXT	false	true	0:3:18.335
	48	95 B
JSON	{type: "peek", peek: {0: "f9f770ac91089346c1b0c0042ca4da1e", 1: "01d120f388c983df2dcbcd4b9c27dd26"}}
	TEXT	false	true	0:3:18.335
	48	716 B
JSON	{cmd: "offer", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	true	true	0:3:18.454
	48	716 B
JSON	{cmd: "offer", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	true	true	0:3:18.470
	48	250 B
JSON	{cmd: "icecandidate", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	true	true	0:3:18.584
	48	274 B
JSON	{cmd: "icecandidate", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	true	true	0:3:18.584
	48	142 B
JSON	{cmd: "icecandidate", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	true	true	0:3:18.584
	48	250 B
JSON	{cmd: "icecandidate", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	true	true	0:3:18.585
	48	274 B
JSON	{cmd: "icecandidate", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	true	true	0:3:18.585
	48	142 B
JSON	{cmd: "icecandidate", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", dstId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	true	true	0:3:18.585
	48	717 B
JSON	{cmd: "answer", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	false	true	0:3:18.636
	48	717 B
JSON	{cmd: "answer", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	false	true	0:3:18.636
	48	251 B
JSON	{cmd: "icecandidate", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	false	true	0:3:18.739
	48	275 B
JSON	{cmd: "icecandidate", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	false	true	0:3:18.739
	48	143 B
JSON	{cmd: "icecandidate", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "f9f770ac91089346c1b0c0042ca4da1e", more...}
	TEXT	false	true	0:3:18.747
	48	251 B
JSON	{cmd: "icecandidate", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	false	true	0:3:18.747
	48	275 B
JSON	{cmd: "icecandidate", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	false	true	0:3:18.747
	48	143 B
JSON	{cmd: "icecandidate", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", srcId: "01d120f388c983df2dcbcd4b9c27dd26", more...}
	TEXT	false	true	0:3:18.747
	48	72 B
JSON	{type: "bind", id: "f9f770ac91089346c1b0c0042ca4da1e", reserved: false}
	TEXT	true	true	0:3:19.484
	48	69 B
JSON	{type: "evaluate", id: "01d120f388c983df2dcbcd4b9c27dd26", value: 0}
	TEXT	true	true	0:3:21.360
	48	Disconnectedcode: 1006
24 frames5.90 KB3.03s
Client1:
	44	16 B
JSON	{type: "getid"}
	TEXT	true	true	0:3:0.443
	44	15 B
JSON	{type: "find"}
	TEXT	true	true	0:3:0.444
	44	54 B
JSON	{id: "f9f770ac91089346c1b0c0042ca4da1e", type: "id"}
	TEXT	false	true	0:3:0.444
	44	65 B
JSON	{cmd: "start", dst: "server", msg: "bind", more...}
	TEXT	false	true	0:3:0.444
	44	34 B
JSON	{type: "peek", peek: {0: "server"}}
	TEXT	false	true	0:3:0.444
	45	Connected to:ws://127.0.0.1:8888/stream/test
	45	419 B
MQTT	{cmd: "connect", retain: true, qos: 2, more...}
	BINARY	false	true	0:3:0.477
	45	4.27 KB
MQTT	{cmd: "connect", retain: true, qos: 2, more...}
	BINARY	false	true	0:3:0.478
	45	5.69 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:0.478
	45	5.72 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:0.478
	45	5.68 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:0.479
	44	46 B
JSON	{type: "bind", id: "server", reserved: false}
	TEXT	true	true	0:3:0.502
	44	35 B
JSON	{cmd: "end", type: "transaction"}
	TEXT	false	true	0:3:0.502
	45	5.98 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:1.503
	45	5.67 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:3.15
	45	5.71 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:3.521
	45	5.95 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:4.553
	45	5.69 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:8.142
	45	5.70 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:8.659
	45	5.68 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:9.679
	45	5.72 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:10.703
	45	5.66 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:11.207
	45	5.69 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:13.769
	45	5.70 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:15.294
	45	5.72 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:15.815
	45	5.72 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:16.328
	44	91 B
JSON	{cmd: "start", dst: "0cdd78b4c007db9a3d5d1700b0111da5", msg: "peek", more...}
	TEXT	false	true	0:3:18.335
	44	717 B
JSON	{cmd: "offer", dstId: "f9f770ac91089346c1b0c0042ca4da1e", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	false	true	0:3:18.471
	44	251 B
JSON	{cmd: "icecandidate", dstId: "f9f770ac91089346c1b0c0042ca4da1e", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	false	true	0:3:18.584
	44	275 B
JSON	{cmd: "icecandidate", dstId: "f9f770ac91089346c1b0c0042ca4da1e", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	false	true	0:3:18.584
	44	143 B
JSON	{cmd: "icecandidate", dstId: "f9f770ac91089346c1b0c0042ca4da1e", srcId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	false	true	0:3:18.585
	44	716 B
JSON	{cmd: "answer", srcId: "f9f770ac91089346c1b0c0042ca4da1e", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	true	true	0:3:18.622
	44	250 B
JSON	{cmd: "icecandidate", srcId: "f9f770ac91089346c1b0c0042ca4da1e", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	true	true	0:3:18.729
	44	274 B
JSON	{cmd: "icecandidate", srcId: "f9f770ac91089346c1b0c0042ca4da1e", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	true	true	0:3:18.739
	44	142 B
JSON	{cmd: "icecandidate", srcId: "f9f770ac91089346c1b0c0042ca4da1e", dstId: "0cdd78b4c007db9a3d5d1700b0111da5", more...}
	TEXT	true	true	0:3:18.739
	44	35 B
JSON	{cmd: "end", type: "transaction"}
	TEXT	false	true	0:3:19.485
	45	5.68 KB
MQTT	{cmd: "reserved", retain: false, qos: 0, more...}
	BINARY	false	true	0:3:19.906
	44	2 B	ê	CLOSE	true	true	0:4:55.644
	45	Disconnectedcode: 1006
	44	Disconnectedcode: 1006
 */