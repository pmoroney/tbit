package main

import "log"

// TODO(pmo): use benchmarks to tune this value or make it a config variable.
const outputBufSize = 100

var Rooms = &RoomList{list: make(map[string]*Room)}

func main() {
	s := NewServer()
	log.Fatal(s.ListenAndServe())
}
