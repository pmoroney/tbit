package main

import (
	"io"
	"log"
	"net"
	"os"

	"github.com/pelletier/go-toml"
)

// TODO(pmo): use benchmarks to tune this value or make it a config variable.
const outputBufSize = 100

type settings struct {
	Host    string
	Port    string
	LogFile string
}

func (s *settings) readConfig(r io.Reader) error {
	d := toml.NewDecoder(r)
	return d.Decode(s)
}

func main() {
	config := settings{
		Host:    "",
		Port:    "9999",
		LogFile: "tbit.log",
	}

	file, err := os.Open("tbit.conf")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("error while reading config file: %s", err)
		}
		log.Print("config file not found, using defaults")
	}
	if file != nil {
		defer file.Close()
		err := config.readConfig(file)
		if err != nil {
			log.Fatalf("fatal error parsing config file: %s", err)
		}
	}

	file, err = os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Fatal error opening log file: %s \n", err)
	}
	defer file.Close()
	log.SetOutput(io.MultiWriter(file, os.Stderr))

	s := NewServer()
	s.Addr = net.JoinHostPort(config.Host, config.Port)

	log.Fatal(s.ListenAndServe())
}
