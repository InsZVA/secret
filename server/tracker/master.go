package main

import (
	"net/http"
	"sync"
	"crypto/md5"
	"time"
	"github.com/gorilla/websocket"
	"encoding/hex"
	"strconv"
	"sort"
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
	if k == "server" { return &server }
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
				if source := cli.inputs[i].output; source != nil {
					t += ":[from]" + source.id
				}
			case INPUT_STATE_RESERVED:
				t += "Reserved"
				if source := cli.inputs[i].output; source != nil {
					t += ":[from]" + source.id
				}
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
				if dst := cli.outputs[i].input; dst != nil {
					t += ":[to]" + dst.id
				}
			case OUTPUT_STATE_RESERVED:
				t += "Reserved"
				if dst := cli.outputs[i].input; dst != nil {
					t += ":[to]" + dst.id
				}
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
	output *Client
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
	input *Client
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

func (cli *Client) Source() *Client {
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

	if cli != &server {
		transactionStartMsg["msg"] = msg
		transactionStartMsg["dst"] = dst.id
		err := cli.conn.WriteJSON(transactionStartMsg)
		if err != nil {
			return false
		}
		cli.state = CLIENT_STATE_IN_TRASACTION
	}
	return true
}

// end both peers' transaction
func (t *Transaction) End() {
	i := 0
	if t.a.transaction != nil && len(t.a.transaction) != 0 {
		for ; i < len(t.a.transaction); i++ {
			if t.a.transaction[i].b == t.b {
				break
			}
		}
		t.a.transaction = append(t.a.transaction[:i], t.a.transaction[i+1:]...)
	}
	if t.b.transaction != nil && len(t.b.transaction) != 0 {
		for i = 0; i < len(t.b.transaction); i++ {
			if t.b.transaction[i].b == t.a {
				break
			}
		}
		t.b.transaction = append(t.b.transaction[:i], t.b.transaction[i+1:]...)
	}

	if t.a.transaction == nil || len(t.a.transaction) == 0 {
		if t.a != &server {
			t.a.TransactionEnd()
		}
	}
	if t.b.transaction == nil || len(t.b.transaction) == 0 {
		if t.b != &server {
			t.b.TransactionEnd()
		}
	}
}

var (
	transactionStartMsg = make(map[string]interface{})
	transactionEndMsg = make(map[string]interface{})
)

func init() {
	transactionStartMsg["type"] = "transaction"
	transactionStartMsg["cmd"] = "start"

	transactionEndMsg["type"] = "transaction"
	transactionEndMsg["cmd"] = "end"
}

/**
	Transaction end:
	client -> server
 */
func (cli *Client) TransactionEnd() {
	if cli != &server {
		cli.state = CLIENT_STATE_READY
		if cli.conn != nil {
			cli.conn.WriteJSON(transactionEndMsg)
		}
	}
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

func (cli *Client) FindTransactions(dst *Client) []int {
	ret := []int{}
	for i := 0; i < len(cli.transaction); i++ {
		if cli.transaction[i].b == dst {
			ret = append(ret, i)
		}
	}
	return ret
}

// TODO: every client's operation must be in its own thread to avoid lock
// TODO: setState notification
func (cli *Client) EndTransactions(ids []int) {
	left := []Transaction{}
	todel := []Transaction{}
	sort.Ints(ids)
	p, i := 0, 0
	for p < len(ids) {
		if i >= len(cli.transaction) {
			break
		}
		if i == ids[p] {
			todel = append(todel, cli.transaction[i])
			i++
			p++
		} else if ids[p] > i {
			left = append(left, cli.transaction[i])
			i++
		} else {
			//impossible
			panic("impossible")
		}
	}
	for _, t := range todel {
		t.End()
	}
	cli.transaction = left // not necessary
}

func (cli *Client) Bind(dst *Client, reserved bool) {
	cli.SetReadyInput(dst, reserved)
	dst.SetReadyOutput(cli, reserved)
	t := cli.FindTransactions(dst)
	cli.EndTransactions(t)
}

// create or get a ready input and set its output
func (cli *Client) SetReadyInput(dst *Client, reserved bool) {
	if dst == nil { return }
	state := INPUT_STATE_RUNNING
	if reserved { state = INPUT_STATE_RESERVED }
	if cli.inputs == nil {
		cli.inputs = []Input{{
			cli: cli, state: state,
			output: dst, level: dst.Level() + 1,
		}}
		return
	}

	for i := 0; i < len(cli.inputs); i++ {
		if cli.inputs[i].state == INPUT_STATE_CLOSE {
			cli.inputs[i].state = state
			cli.inputs[i].output = dst
			cli.inputs[i].level = dst.Level() + 1
			return
		}
	}

	cli.inputs = append(cli.inputs, Input{
		cli: cli, state: state,
		output: dst, level: dst.Level() + 1,
	})
}

// create or get a ready output and set its input
// if dst is nil, this function only ensure a ready output
func (cli *Client) SetReadyOutput(dst *Client, reserved bool) {
	state := OUTPUT_STATE_RUNNING
	if reserved { state = OUTPUT_STATE_RESERVED }
	if dst == nil { state = OUTPUT_STATE_READY }
	if cli.outputs == nil {
		cli.outputs = []Output{{
			cli: cli, state: state,
			input: dst,
		}}
	} else {
		closeId := -1
		for i := 0; i < len(cli.outputs); i++ {
			if cli.outputs[i].state == OUTPUT_STATE_READY {
				cli.outputs[i].state = state
				cli.outputs[i].input = dst
				return
			}
			if cli.outputs[i].state == OUTPUT_STATE_CLOSE {
				closeId = i
			}
		}

		//no ready, change closeId to ready
		if closeId != -1 {
			cli.outputs[closeId].state = state
			cli.outputs[closeId].input = dst
			return
		}

		cli.outputs = append(cli.outputs, Output{
			cli: cli, state: state,
			input: dst,
		})
	}
}

func (cli *Client) Evaluate(dst *Client, value bool) {
	if value {
		dst.SetReadyOutput(nil, false)
	} else {
		if i := cli.FindTransaction(dst); i != -1 {
			cli.transaction[i].End()
		}
	}
}

const (
	SERVER_OUTPUT_CAPABILITY = 2
)

func (cm *ClientMap) PeekLevel(cli *Client, l int) []*Client {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	peeked := []*Client{}
	i := 0
	for _, c := range cm.m {
		if c.id == cli.id { continue }
		if c.Level() - l < 2 && c.Level() - 1 > -2 &&
			c.state != CLIENT_STATE_IN_TRASACTION {
			if c.OutputAvailable() || i < 2 {
				// i < 2 means randomly peek 2 client to evaluate
				peeked = append(peeked, c)
				if len(peeked) > 5 {
					return peeked
				}
			}
		}
		i++
	}
	return peeked
}

func (cli *Client) Find() []*Client {
	level := cli.Level()
	if level < 1 {
		if clientMap.n <= SERVER_OUTPUT_CAPABILITY {
			return []*Client{&server}
		}
		level = maxLevel / 2 + 1
	}
	return clientMap.PeekLevel(cli, level)
}

// Free some output of specify state
// all outputs of the same state is the same
// param: table[state] = num
func (cli *Client) FreeOutputs(table []int) {
	for i := 0; i < len(cli.outputs); i++ {
		for state := range table {
			if table[state] > 0 && cli.outputs[i].state == state {
				cli.outputs[i].state = OUTPUT_STATE_READY
				table[state]--
			}
		}
	}
}

func (cli *Client) FreeOutput(state int) {
	for i := 0; i < len(cli.outputs); i++ {
		if cli.outputs[i].state == state {
			cli.outputs[i].state = OUTPUT_STATE_READY
			cli.outputs[i].input = nil
			return
		}
	}
}

func state_in2out(in int) int {
	switch in {
	case INPUT_STATE_RESERVED:
		return OUTPUT_STATE_RESERVED
	case INPUT_STATE_RUNNING:
		return OUTPUT_STATE_RUNNING
	}
	return 0
}

func state_out2in(out int) int {
	switch out {
	case OUTPUT_STATE_RUNNING:
		return INPUT_STATE_RUNNING
	case OUTPUT_STATE_RESERVED:
		return INPUT_STATE_RESERVED
	}
	return 0
}

// Release all inputs of a client
func (cli *Client) ReleaseInputs() {
	for i := 0; i < len(cli.inputs); i++ {
		if cli.inputs[i].state == INPUT_STATE_RUNNING ||
			cli.inputs[i].state == INPUT_STATE_RESERVED {
			cli.inputs[i].output.FreeOutput(state_in2out(cli.inputs[i].state))
		}
	}
}

// Close a input of a specify state
func (cli *Client) CloseInput(state int) {
	for i := 0; i < len(cli.inputs); i++ {
		if cli.inputs[i].state == state {
			cli.inputs[i].state = INPUT_STATE_CLOSE
			cli.inputs[i].output = nil
			return
		}
	}
}

// Close all outputs of a client
func (cli *Client) CloseOutputs() {
	for i := 0; i < len(cli.outputs); i++ {
		if cli.outputs[i].state == OUTPUT_STATE_RESERVED ||
			cli.outputs[i].state == OUTPUT_STATE_RUNNING {
			cli.outputs[i].input.CloseInput(state_out2in(cli.outputs[i].state))
		}
	}
}

func (cli *Client) Close() {
	ids := []int{}
	if cli.transaction != nil {
		for i := 0; i < len(cli.transaction); i++ {
			ids = append(ids, i)
		}
	}
	cli.EndTransactions(ids)

	// Release Input & Output
	cli.CloseOutputs()
	cli.ReleaseInputs()
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
	/*defer func() {
		if e := recover(); e != nil {
			err := make(map[string]interface{})
			err["type"] = "error"
			err["msg"] = e
			conn.WriteJSON(err)
		}
	} ()*/

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
			peeked[i].TransactionBegin(cli, "peek")
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
	case "forward":
		dstid, ok := data["dstId"].(string)
		if !ok {
			panic("param type error")
		}
		dst := clientMap.Get(dstid)
		if dst == nil {
			// TODO
		}
		//TODO: control single thread to produce
		dst.conn.WriteJSON(data)
	}
	if len(ret) != 0 {
		conn.WriteJSON(ret)
	}
}