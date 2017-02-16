package main

import (
	"net/http"
	"sync"
	"crypto/md5"
	"time"
	"github.com/gorilla/websocket"
	"encoding/hex"
	"strconv"
)

const (
	CLIENT_STATE_READY = iota
	CLIENT_STATE_IN_TRASACTION
	CLIENT_STATE_SERVER
)

var (
	clientMap = ClientMap{
		m: make(map[string]*Client),
	}
)

//TODO: change to lock-free
type ClientMap struct {
	m map[string]*Client
	lock sync.RWMutex
	n uint32
}

func (cm *ClientMap) Set(k string, v *Client) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	cm.m[k] = v
	cm.n++
}

func (cm *ClientMap) Get(k string) *Client {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	return cm.m[k]
}

func (cm *ClientMap) Remove(k string) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	delete(cm.m, k)
	cm.n--
}

type Transaction struct {
	a *Client
	b *Client
	msg string
}

type Client struct {
	id string
	state int
	inputs []Input
	outputs []Output
	addr string
	conn *websocket.Conn
	evaluated bool
	transaction []Transaction
}

func (cli *Client) InfoHTML() string {
	t := "<table><tr><td>ID</td><td>"
	t += cli.id + "</td></tr><tr><td>"
	t += "State</td><td>"
	switch cli.state {
	case CLIENT_STATE_READY:
		t += "Ready"
	case CLIENT_STATE_IN_TRASACTION:
		t += "Transaction"
	case CLIENT_STATE_SERVER:
		t += "Server"
	}
	t += "</td></tr><tr><td>Inputs</td><td>"
	if cli.inputs != nil {
		t += "<ul>"
		for i, in := range cli.inputs {
			t += "<li>Input#" + strconv.Itoa(i) + ":"
			switch in.state {
			case INPUT_STATE_RUNNING:
				t += "Running"
			case INPUT_STATE_RESERVED:
				t += "Reserved"
			case INPUT_STATE_CLOSE:
				t += "Close"
			}
			t += "</li>"
		}
		t += "</ul>"
	}
	t += "</td></tr><tr><td>Outputs</td><td>"
	if cli.outputs != nil {
		t += "<ul>"
		for i, ou := range cli.outputs {
			t += "<li>Output#" + strconv.Itoa(i) + ":"
			switch ou.state {
			case OUTPUT_STATE_RUNNING:
				t += "Running"
			case OUTPUT_STATE_RESERVED:
				t += "Reserved"
			case OUTPUT_STATE_CLOSE:
				t += "Close"
			case OUTPUT_STATE_READY:
				t += "Ready"
			}
			t += "</li>"
		}
		t += "</ul>"
	}
	t += "</td></tr><tr><td>Address</td><td>"
	t += cli.addr + "</td></tr><tr><td>Transactions:</td><td>"
	if cli.transaction != nil {
		t += "<ul>"
		for i, tr := range cli.transaction {
			t += "<li>Transaction#" + strconv.Itoa(i) + ":"
			t += tr.msg + " <a href='/client/" + tr.a.id + "'>"
			t += tr.a.id + "</a> ---> <a href='/client/" + tr.b.id
			t += "'>" + tr.b.id + "</a>" + "</li>"
		}
		t += "</ul>"
	}
	t += "</td></tr></table>"
	return t
}

var maxLevel = 0

var server = Client {
	id: "server",
	state: CLIENT_STATE_SERVER,
	addr: "0.0.0.0:0",
}

func init() {
	server.ExtendOutput(SERVER_OUTPUT_CAPABILITY)
}

const (
	INPUT_STATE_RUNNING = iota
	INPUT_STATE_RESERVED
	INPUT_STATE_CLOSE
)

/**
	When a input is in-progress(not stable), it will not be the Input
	this Input represents the final-selected input & the transaction state
	will be shown in Client struct
 */
type Input struct {
	cli *Client
	state int
	level int
	output *Output
}

const (
	OUTPUT_STATE_READY = iota
	OUTPUT_STATE_RUNNING
	OUTPUT_STATE_RESERVED
	OUTPUT_STATE_CLOSE
)

/**
	When a output is in-progress(not stable), it will not be the Output
	this Output represents the final-selected input & the transaction state
	will be shown in Client struct
 */
type Output struct {
	cli *Client
	state int
	input *Input
}

// only for server
func (cli *Client) ExtendOutput(n int) {
	if cli.outputs == nil {
		cli.outputs = make([]Output, n)
		for i := 0; i < n; i++ {
			cli.outputs[i].cli = cli
		}
		return
	}

	i := 0
	for i = 0; i < len(cli.outputs); i++ {
		if cli.outputs[i].state == OUTPUT_STATE_CLOSE {
			cli.outputs[i].state = OUTPUT_STATE_READY
			cli.outputs[i].input = nil
			n--
		}
	}

	cli.outputs = append(cli.outputs, make([]Output, n)...)
	for ; i < len(cli.outputs); i++ {
		cli.outputs[i].cli = cli
	}
}

/**
	return 0 means no input
	return -1 means no running input
 */
func (cli *Client) Level() int {
	if cli.inputs == nil {
		return 0
	}
	for _, in := range cli.inputs {
		if in.state == INPUT_STATE_RUNNING {
			return in.level + 1
		}
	}
	return -1
}

func (cli *Client) InputCap() int {
	ret := 0
	if cli.inputs != nil {
		for _, in := range cli.inputs {
			if in.state == INPUT_STATE_RUNNING ||
				in.state == INPUT_STATE_RESERVED {
				ret++
			}
		}
	}
	return ret
}

func (cli *Client) Source() *Output {
	if cli.inputs == nil {
		return nil
	}

	for _, in := range cli.inputs {
		if in.state == INPUT_STATE_RUNNING {
			return in.output
		}
	}
	return nil
}

func (cli *Client) OutputAvailable() bool {
	if cli.outputs == nil {
		return true // 2 available
	}

	if cli.evaluated {
		return cli.OutputCap() - cli.Output() > 0
	}

	return true
}

/**
	the max output capability (if not used before,
	2 is evaluated)
 */
func (cli *Client) OutputCap() int {
	if cli.outputs != nil {
		return len(cli.outputs)
	}
	return 2
}

/**
	the output capability used already (reserved or
	running)
 */
func (cli *Client) Output() int {
	ret := 0
	if cli.outputs != nil {
		for _, ou := range cli.outputs {
			if ou.state == OUTPUT_STATE_RUNNING ||
				ou.state == OUTPUT_STATE_RESERVED {
				ret++
			}
		}
	}
	return ret
}

/**
	Transaction begin:
	server -> client
 */
func (cli *Client) TransactionBegin(dst *Client, msg string) bool {
	if cli.transaction == nil {
		cli.transaction = []Transaction{
			{
				a: cli,
				b: dst,
				msg: msg,
			},
		}
	} else {
		cli.transaction = append(cli.transaction, Transaction{
			a: cli,
			b: dst,
			msg: msg,
		})
	}

	data := make(map[string]interface{})
	data["type"] = "transaction"
	data["msg"] = msg
	err := cli.conn.WriteJSON(data)
	if err != nil {
		return false
	}
	cli.state = CLIENT_STATE_IN_TRASACTION
	return true
}

func (t *Transaction) End() {
	i := 0
	if t.a.transaction != nil {
		for ; i < len(t.a.transaction); i++ {
			if t.a.transaction[i].b == t.b {
				break
			}
		}
		t.a.transaction = append(t.a.transaction[:i], t.a.transaction[i+1:]...)
	}
	if t.b.transaction != nil {
		for i = 0; i < len(t.b.transaction); i++ {
			if t.b.transaction[i].a == t.a {
				break
			}
		}
		t.b.transaction = append(t.b.transaction[:i], t.b.transaction[i+1:]...)
	}
}

/**
	Transaction end:
	client -> server
 */
func (cli *Client) TransactionEnd() {
	cli.state = CLIENT_STATE_READY
}

func (cli *Client) FindTransaction(dst *Client) int {
	ret := 0
	for ; ret < len(cli.transaction); ret++ {
		if cli.transaction[ret].b == dst {
			return ret
		}
	}
	return -1
}

func (cli *Client) Bind(dst *Client, reserved bool) {
	if cli.inputs == nil {
		cli.inputs = []Input{
			Input{
				cli:   cli,
				level: dst.Level() + 1,
			},
		}

		if reserved {
			cli.inputs[0].state = INPUT_STATE_RESERVED
			//cli.inputs[0].output
		}
	} else {
		//TODO
	}
	dst.TransactionEnd()
	cli.TransactionEnd()
}

func (cli *Client) Evaluate(dst *Client, value bool) {
	if i := cli.FindTransaction(dst); i != -1 {
		if value {
			if cli.outputs == nil {
				cli.outputs = []Output{
					{
						cli: cli,
						state: OUTPUT_STATE_READY,
					},
				}
			} else {
				cli.outputs = append(cli.outputs, Output{
					cli: cli,
					state: OUTPUT_STATE_READY,
				})
			}
		}
		cli.transaction[i].End()
	}
}

const (
	SERVER_OUTPUT_CAPABILITY = 2
)

func (cm *ClientMap) PeekLevel(l int) []*Client {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	peeked := []*Client{}
	i := 0
	for _, c := range cm.m {
		if c.Level() - l < 2 && c.Level() - 1 > -2 &&
			c.state != CLIENT_STATE_IN_TRASACTION {
			if c.OutputAvailable() || i < 2 {
				c.state = CLIENT_STATE_IN_TRASACTION
				peeked = append(peeked, c)
			}
			if len(peeked) > 5 {
				return peeked
			}
		}
	}
	return peeked
}

func (cli *Client) Find() []*Client {
	cli.state = CLIENT_STATE_IN_TRASACTION
	level := cli.Level()
	if level < 1 {
		if clientMap.n < SERVER_OUTPUT_CAPABILITY {
			return []*Client{&server}
		}
		level = maxLevel / 2 + 1
	}
	return clientMap.PeekLevel(level)
}

func (cli *Client) Close() {
	if cli.transaction == nil { return }
	if cli.transaction != nil && len(cli.transaction) == 0 {
		cli.transaction = nil
		return
	}
	cli.transaction[0].End()
	cli.Close()
}

func MasterHandler(path []string, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(500); return
	}

	// Register a new Client
	h := md5.New()
	h.Write([]byte(r.RemoteAddr))
	h.Write([]byte(time.Now().String()))
	cli := &Client {
		id: hex.EncodeToString(h.Sum(nil)),
		state: CLIENT_STATE_READY,
		addr: r.RemoteAddr,
		conn: conn,
	}
	clientMap.Set(cli.id, cli)
	defer func() {
		cli.Close()
		clientMap.Remove(cli.id)
	} ()

	data := make(map[string]interface{})
	for conn.ReadJSON(&data) == nil {
		cli.translateMsg(data, conn)
		data = make(map[string]interface{})
	}
}

func (cli *Client) translateMsg(data map[string]interface{}, conn *websocket.Conn) {
	defer func() {
		if e := recover(); e != nil {
			err := make(map[string]interface{})
			err["type"] = "error"
			err["msg"] = e
			conn.WriteJSON(err)
		}
	} ()

	tp, ok := data["type"]
	if !ok {
		panic("type field missing!")
	}

	ret := make(map[string]interface{})
	switch tp {
	case "getid":
		ret["type"] = "id"
		ret["id"] = cli.id
	case "find":
		peeked := cli.Find()
		ret["type"] = "peek"
		ret["peek"] = []string{}
		for i := 0; i < len(peeked); i++ {
			cli.TransactionBegin(peeked[i],"bind")
			ret["peek"] = append(ret["peek"].([]string), peeked[i].id)
		}
	case "bind":
		id, ok := data["id"].(string)
		if !ok {
			panic("param type error")
		}
		dst := clientMap.Get(id)
		if dst == nil {
			// TODO
		}
		reserved, ok := data["reserved"].(bool)
		if !ok {
			panic("param type error")
		}
		cli.Bind(dst, reserved)
	case "evaluate":
		id, ok := data["id"].(string)
		if !ok {
			panic("param type error")
		}
		dst := clientMap.Get(id)
		if dst == nil {
			// TODO
		}
		value, ok := data["value"].(float64)
		if !ok {
			panic("param type error")
		}
		cli.Evaluate(dst, value != 0)
	}
	conn.WriteJSON(ret)
}