package main

import (
	"errors"
	"log"
	"net"
	"sort"
	"sync"
)

// Server controls the room list as well as username list.
type Server struct {
	Addr string

	rooms     *roomList
	usernames *usernameList
}

// NewServer creates a new server
func NewServer() *Server {
	return &Server{
		rooms: &roomList{
			list: make(map[string]*Room),
		},
		usernames: &usernameList{
			usernameToID: make(map[string]int),
			idToUsername: make(map[int]string),
		},
	}
}

// ListenAndServe listens on `Addr` and spawns connections in their own goroutine.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	log.Printf("Listening on %s\n", s.Addr)
	defer ln.Close()
	id := 1
	s.rooms.create("lobby")
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		// log the RemoteAddr here because NewConn() stores it as a io.ReadWriteCloser
		log.Printf("New connection id %d from %s\n", id, conn.RemoteAddr().String())
		c := s.NewConn(conn, id)
		if c == nil {
			conn.Close()
			continue
		}
		go c.handleConnection()
		id++
	}
}

// roomList encapsulates the list of rooms
type roomList struct {
	sync.RWMutex
	list map[string]*Room
}

// create creates a new room
func (rl *roomList) create(name string) *Room {
	rl.Lock()
	defer rl.Unlock()
	r := NewRoom(name)
	rl.list[name] = r
	return r
}

// get returns the named room
func (rl *roomList) get(name string) *Room {
	rl.RLock()
	defer rl.RUnlock()
	r, ok := rl.list[name]
	if !ok {
		return nil
	}
	return r
}

// listAll returns a list of all the room names
func (rl *roomList) listAll() []string {
	rl.RLock()
	defer rl.RUnlock()
	list := make([]string, 0, len(rl.list))
	for r := range rl.list {
		list = append(list, r)
	}
	sort.Strings(list)
	return list
}

// usernameList encapsulates the mapping of id to username and visa versa.
type usernameList struct {
	sync.RWMutex
	usernameToID map[string]int
	idToUsername map[int]string
}

// getUsername returns the username for the connection id
func (ul *usernameList) getUsername(id int) string {
	ul.RLock()
	defer ul.RUnlock()
	return ul.idToUsername[id]
}

// addUsername creates a username for a new connection id
func (ul *usernameList) addUsername(id int, name string) error {
	ul.Lock()
	defer ul.Unlock()

	_, exists := ul.idToUsername[id]
	if exists {
		return errors.New("connection already has a username")
	}

	_, exists = ul.usernameToID[name]
	if exists {
		return errors.New("username already exists")
	}

	ul.idToUsername[id] = name
	ul.usernameToID[name] = id
	return nil
}

// removeUsername removes the username for a connection id. Used for disconnecting connections.
func (ul *usernameList) removeUsername(id int) error {
	ul.Lock()
	defer ul.Unlock()

	oldName, ok := ul.idToUsername[id]
	if !ok {
		return errors.New("connection does not have a username already")
	}

	delete(ul.usernameToID, oldName)
	delete(ul.idToUsername, id)
	return nil
}

// modifyUsername changes a username for a connection.
func (ul *usernameList) modifyUsername(id int, name string) error {
	ul.Lock()
	defer ul.Unlock()

	_, exists := ul.usernameToID[name]
	if exists {
		return errors.New("username already exists")
	}

	oldName, ok := ul.idToUsername[id]
	if !ok {
		return errors.New("connection does not have a username already")
	}

	if oldName == name {
		return nil
	}

	ul.idToUsername[id] = name
	ul.usernameToID[name] = id
	delete(ul.usernameToID, oldName)

	return nil
}
