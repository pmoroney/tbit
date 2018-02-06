package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
)

var helpText = `Welcome to Tbit chat!
Commands:
/exit
/help
/quit
/user <username>
`

var welcomeText = `Welcome to Tbit chat!
Type /help for a list of commands.
Your username is currently: %s
Use the "/user <username>" command to change it
`

type Conn struct {
	c          io.ReadWriteCloser
	server     *Server
	id         int
	username   string
	outputChan chan string
	closeChan  chan struct{}
	rooms      map[string]bool
}

// NewConn creates a Conn.
func (s *Server) NewConn(c io.ReadWriteCloser, id int) *Conn {
	conn := &Conn{
		c:          c,
		server:     s,
		id:         id,
		username:   fmt.Sprintf("Anonymous%d", id),
		outputChan: make(chan string, outputBufSize),
		closeChan:  make(chan struct{}),
		rooms:      make(map[string]bool),
	}
	err := conn.server.usernames.addUsername(conn.id, conn.username)
	if err != nil {
		log.Println(err)
		return nil
	}
	conn.JoinRoom("lobby")
	return conn
}

func (c Conn) inRoom(roomName string) bool {
	// don't have to check ', ok' since if ok is false, then inRoom would be as well.
	inRoom := c.rooms[roomName]
	return inRoom
}

func (c *Conn) listRooms() []string {
	list := make([]string, 0, len(c.rooms))
	for r, inRoom := range c.rooms {
		if !inRoom {
			continue
		}
		list = append(list, r)
	}
	sort.Strings(list)
	return list
}

func (c *Conn) JoinRoom(roomName string) {
	r := c.server.rooms.Get(roomName)
	if r == nil {
		r = c.server.rooms.Create(roomName)
	}
	r.Join(c)
	// This is only called from the goroutine for Conn.handleConnection() so locking c.rooms is not nessisary.
	c.rooms[roomName] = true
}

func (c *Conn) LeaveRoom(roomName string) error {
	if !c.inRoom(roomName) {
		return errors.New("you are not currently in that room")
	}
	c.rooms[roomName] = false
	r := c.server.rooms.Get(roomName)
	if r == nil {
		return errors.New("you were in a room that did not exist")
	}
	r.Announce(fmt.Sprintf("%s has left the room", c.Username()), "server")
	r.Leave(c)
	return nil
}

func (c Conn) Username() string {
	return c.username
}

func (c *Conn) Announce(msg string) {
	// TODO(pmo): Allow users to set their timezone. This will require setting the timestamp in the receiving Conn.
	// This is only called from the goroutine for Conn.handleConnection() so locking c.rooms is not nessisary.
	for r, inRoom := range c.rooms {
		if inRoom {
			c.server.rooms.Get(r).Announce(msg, c.Username())
		}
	}
}

func (c *Conn) handleMessages() {
	for {
		select {
		case msg := <-c.outputChan:
			io.WriteString(c.c, msg)
		case <-c.closeChan:
			return
		}
	}
}

func (c *Conn) Close() error {
	c.closeChan <- struct{}{}
	for r, inRoom := range c.rooms {
		if inRoom {
			c.LeaveRoom(r)
		}
	}
	err := c.server.usernames.removeUsername(c.id)
	if err != nil {
		log.Printf("error while closing connection: %s\n", err)
	}
	return c.c.Close()
}

func (c *Conn) handleConnection() {
	defer c.Close()

	fmt.Fprintf(c.c, welcomeText, c.Username())

	go c.handleMessages()

	scanner := bufio.NewScanner(c.c)
	for scanner.Scan() {
		input := scanner.Text()

		if input == "" {
			continue
		}

		log.Print(c.Username(), ": ", input)
		if input[0] == '/' {
			if !c.handleCommand(input) {
				return
			}
			continue
		}
		c.Announce(input)
	}
	if err := scanner.Err(); err != nil {
		log.Print("error scanning lines:", err)
	}
}

// handleCommand performs the actions of a /command.
// handleCommand returns false if handleConnection is to quit
func (c *Conn) handleCommand(input string) bool {
	fields := strings.Fields(input)
	switch fields[0] {
	case "/help":
		io.WriteString(c.c, helpText)
	case "/exit", "/quit":
		log.Printf("%s has disconnected\n", c.Username())
		return false
	case "/user":
		if len(fields) != 2 {
			fmt.Fprintln(c.c, "Usage is /user <username>")
			return true
		}
		if fields[1] == "server" {
			fmt.Fprintln(c.c, "Username cannot be 'server'")
			return true
		}
		err := c.server.usernames.modifyUsername(c.id, fields[1])
		if err != nil {
			fmt.Fprintln(c.c, err)
			return true
		}
		c.username = fields[1]
	case "/join":
		if len(fields) != 2 {
			fmt.Fprintln(c.c, "Usage is /join <room>")
			return true
		}
		c.JoinRoom(fields[1])
	case "/leave":
		if len(fields) != 2 {
			fmt.Fprintln(c.c, "Usage is /leave <room>")
			return true
		}
		err := c.LeaveRoom(fields[1])
		if err != nil {
			fmt.Fprintln(c.c, err)
		}
	case "/rooms":
		fmt.Fprintln(c.c, "Here is a list of the current rooms:")
		for _, r := range c.server.rooms.ListAll() {
			fmt.Fprintln(c.c, r)
		}
		// Output an empty line so the client has a way to know if the list has ended.
		fmt.Fprintln(c.c, "")
	case "/list":
		fmt.Fprintln(c.c, "You are in the following rooms:")
		for _, r := range c.listRooms() {
			fmt.Fprintln(c.c, r)
		}
		// Output an empty line so the client has a way to know if the list has ended.
		fmt.Fprintln(c.c, "")
	case "/say":
		if len(fields) < 3 {
			fmt.Fprintln(c.c, "Usage is /say <room> <message>")
			return true
		}
		if !c.inRoom(fields[1]) {
			fmt.Fprintln(c.c, "You are not in that room")
			return true
		}
		r := c.server.rooms.Get(fields[1])
		if r == nil {
			c.LeaveRoom(fields[1])
			fmt.Fprintln(c.c, "You were in a room that did not exist")
			return true
		}
		// Find the index of the first message field of the input.
		// This way we don't loose the whitespace of the message.
		i := strings.Index(input, fields[3])
		msg := input[i:]
		r.Announce(msg, c.Username())
	default:
		fmt.Fprintf(c.c, "Unknown command: %s\n", fields[0])
	}
	return true
}
