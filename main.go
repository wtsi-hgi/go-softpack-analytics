package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var port, server uint64

	flag.Uint64Var(&port, "p", 1234, "port to listen on for analytics")
	flag.Uint64Var(&server, "l", 12345, "port to listen on for web server")

	flag.Parse()

	al, err := net.ListenTCP("tcp", &net.TCPAddr{
		Port: int(port),
	})
	if err != nil {
		return err
	}

	sl, err := net.ListenTCP("tcp", &net.TCPAddr{
		Port: int(server),
	})
	if err != nil {
		return err
	}

	return newAnalyticsServer(al, sl)
}

func newAnalyticsServer(al, sl *net.TCPListener) error {
	// TODO: Start http server

	for {
		c, err := al.AcceptTCP()
		if err != nil {
			return err
		}

		go handleAnalytics(c)
	}
}

type Analytic struct {
	Name, Command, IP string
	Time              int64
}

func handleAnalytics(c *net.TCPConn) {
	var sb strings.Builder

	if _, err := io.Copy(&sb, io.LimitReader(c, 4096)); err != nil {
		return
	}

	parts := strings.Split(sb.String(), "\x00")

	if len(parts) != 2 {
		return
	}

	ra := c.RemoteAddr().(*net.TCPAddr)

	fmt.Println(Analytic{
		Name:    parts[0],
		Command: parts[1],
		IP:      ra.IP.String(),
		Time:    time.Now().Unix(),
	})
}
