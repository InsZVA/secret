// +build !product

package main

import "log"

func debug(v interface{}) {
	log.Println(v)
}