package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Room struct {
	sync.RWMutex
	Name  string
	Conns map[int]*Conn
}

func NewRoom(name string) *Room {
	return &Room{
		Conns: make(map[int]*Conn),
		Name:  name,
	}
}

func (r *Room) Join(conn *Conn) {
	r.Lock()
	defer r.Unlock()
	r.Conns[conn.id] = conn
}

func (r *Room) Leave(conn *Conn) {
	r.Lock()
	defer r.Unlock()
	delete(r.Conns, conn.id)
}

func (r *Room) Announce(msg, username string) {
	// TODO(pmo): Allow users to set their timezone. This will require setting the timestamp in the receiving Conn.
	roomMsg := fmt.Sprintf("%s %s %s: %s\n", time.Now().Format(time.RFC3339), r.Name, username, msg)
	log.Printf("%s %s: %s\n", r.Name, username, msg)
	r.RLock()
	defer r.RUnlock()
	for id, conn := range r.Conns {
		select {
		case conn.outputChan <- roomMsg:
		case <-time.After(1 * time.Second):
			// TODO(pmo): tune this timeout and/or add to config variables.
			log.Printf("timeout sending to outputChan %d", id)
		}
	}
}
