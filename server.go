package main

import (
	"errors"
	"log"
	"net"
	"sort"
	"sync"
)

type Server struct {
	rooms     *RoomList
	usernames *UsernameList
}

func NewServer() *Server {
	return &Server{
		rooms: &RoomList{
			list: make(map[string]*Room),
		},
		usernames: &UsernameList{
			usernameToID: make(map[string]int),
			idToUsername: make(map[int]string),
		},
	}
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", ":9999")
	if err != nil {
		return err
	}
	defer ln.Close()
	id := 1
	Rooms.Create("lobby")
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

type RoomList struct {
	sync.RWMutex
	list map[string]*Room
}

func (rl *RoomList) Create(name string) *Room {
	rl.Lock()
	defer rl.Unlock()
	r := &Room{
		Conns: make(map[int]*Conn),
		Name:  name,
	}
	rl.list[name] = r
	return r
}

func (rl *RoomList) Get(name string) *Room {
	rl.RLock()
	defer rl.RUnlock()
	r, ok := rl.list[name]
	if !ok {
		return nil
	}
	return r
}

func (rl *RoomList) ListAll() []string {
	rl.RLock()
	defer rl.RUnlock()
	list := make([]string, 0, len(rl.list))
	for r := range rl.list {
		list = append(list, r)
	}
	sort.Strings(list)
	return list
}

type UsernameList struct {
	sync.RWMutex
	usernameToID map[string]int
	idToUsername map[int]string
}

func (ul *UsernameList) getUsername(id int) string {
	ul.RLock()
	defer ul.RUnlock()
	return ul.idToUsername[id]
}

func (ul *UsernameList) addUsername(id int, name string) error {
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

func (ul *UsernameList) removeUsername(id int) error {
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

func (ul *UsernameList) modifyUsername(id int, name string) error {
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
