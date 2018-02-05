package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// TODO(pmo): use benchmarks to tune this value or make it a config variable.
const outputBufSize = 100

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

type ChannelList struct {
	sync.RWMutex
	list map[string]*Channel
}

func (cl *ChannelList) Create(name string) {
	cl.Lock()
	defer cl.Unlock()
	cl.list[name] = &Channel{Conns: make(map[int]*Conn)}
}

func (cl *ChannelList) Get(name string) *Channel {
	cl.RLock()
	defer cl.RUnlock()
	ch, ok := cl.list[name]
	if !ok {
		return nil
	}
	return ch
}

var Channels = &ChannelList{list: make(map[string]*Channel)}

type Channel struct {
	sync.RWMutex
	Conns map[int]*Conn
}

func (ch *Channel) Join(conn *Conn) {
	ch.Lock()
	defer ch.Unlock()
	ch.Conns[conn.id] = conn
}

func (ch *Channel) Leave(conn *Conn) {
	ch.Lock()
	defer ch.Unlock()
	delete(ch.Conns, conn.id)
}

func (ch *Channel) Announce(msg string) {
	ch.RLock()
	defer ch.RUnlock()
	for id, conn := range ch.Conns {
		select {
		case conn.outputChan <- msg:
		case <-time.After(1 * time.Second):
			// TODO(pmo): tune this timeout and/or add to config variables.
			log.Printf("timeout sending to outputChan %d", id)
		}
	}
}

type Conn struct {
	c          net.Conn
	id         int
	username   string
	outputChan chan string
	closeChan  chan struct{}
	channel    string
}

func NewConn(c net.Conn, id int) *Conn {
	conn := &Conn{
		c:          c,
		id:         id,
		username:   fmt.Sprintf("Anonymous%d", id),
		outputChan: make(chan string, outputBufSize),
		closeChan:  make(chan struct{}),
	}
	Channels.Get("").Join(conn)
	return conn
}

func (c Conn) Username() string {
	return c.username
}

func (c *Conn) Announce(msg string) {
	// TODO(pmo): Allow users to set their timezone. This will require setting the timestamp in the receiving Conn.
	msg = fmt.Sprintf("%s %s: %s\n", time.Now().Format(time.RFC3339), c.Username(), msg)
	Channels.Get(c.channel).Announce(msg)
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
			fields := strings.Fields(input)
			switch fields[0] {
			case "/help":
				io.WriteString(c.c, helpText)
			case "/exit", "/quit":
				log.Printf("%s has disconnected\n", c.Username())
				return
			case "/die":
				// TODO(pmo): remove this if the service is ever used in production
				// This is just a convenience for testing
				if strings.HasPrefix(c.c.RemoteAddr().String(), "127.0.0.1") {
					log.Fatalf("%s from %s has requested to exit", c.Username(), c.c.RemoteAddr().String())
				}
			case "/user":
				if len(fields) != 2 {
					fmt.Fprintln(c.c, "Incorrect number of arguments")
					continue
				}
				c.username = fields[1]
			default:
				fmt.Fprintf(c.c, "Unknown command: %s\n", fields[0])
			}
			continue
		}
		c.Announce(input)
	}
	if err := scanner.Err(); err != nil {
		log.Print("error scanning lines:", err)
	}
}

func main() {
	ln, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	id := 1
	Channels.Create("")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		c := NewConn(conn, id)
		go c.handleConnection()
		id++
	}
}
