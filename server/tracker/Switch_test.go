package main

import (
	"testing"
	"log"
)

func TestSwitch_AWait(t *testing.T) {
	s := Switch{}
	log.Println(s.Inited())
	t.Error(s.Inited())
	return
	s.Reset()
	go func() {
		for {
			s.AWait()
			log.Println("A")
			s.ARelease()
		}
	} ()
	for {
		s.BWait()
		log.Println("B")
		s.BRelease()
	}
}
