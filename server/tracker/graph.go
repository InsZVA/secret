package main

import (
	"net/http"
	"io/ioutil"
	"math/rand"
	"strconv"
	"os/exec"
	"os"
	"bytes"
	"log"
)

var (
	DOTPATH = "dot"
)

func GraphHandler(path []string, w http.ResponseWriter, r *http.Request) {
	dotTXT := clientMap.GraphString()
	fn := "./" + strconv.Itoa(rand.Int())
	log.Println(dotTXT)
	ioutil.WriteFile(fn + ".dot", []byte(dotTXT), 0777)
	defer os.Remove(fn + ".dot")
	outbuf := bytes.Buffer{}
	errbuf := bytes.Buffer{}
	cmd := exec.Command(DOTPATH, "-Tjpg", fn + ".dot", "-o", fn + ".jpg")
	cmd.Stdout, cmd.Stderr = &outbuf, &errbuf
	cmd.Run()
	log.Println(string(outbuf.Bytes()), string(errbuf.Bytes()))
	defer os.Remove(fn + ".jpg")
	b, err := ioutil.ReadFile(fn + ".jpg")
	if err != nil {
		w.Write([]byte(err.Error())); return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(b)
}