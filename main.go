/*******************************************************************************
 * Copyright (c) 2023 Genome Research Ltd.
 *
 * Authors:
 *	- Sendu Bala <sb10@sanger.ac.uk>
 *	- Michael Woolnough <mw31@sanger.ac.uk>
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 ******************************************************************************/

package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
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
	var (
		port   uint64
		output string
	)

	flag.Uint64Var(&port, "p", 1234, "port to listen on for analytics")
	flag.StringVar(&output, "o", "-", "output file (- is STDOUT)")

	flag.Parse()

	al, err := net.ListenTCP("tcp", &net.TCPAddr{
		Port: int(port),
	})
	if err != nil {
		return err
	}

	var f *os.File

	if output == "-" {
		f = os.Stdout
	} else {
		f, err = os.OpenFile(output, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
	}

	defer f.Close()

	return newAnalyticsServer(al, f)
}

func newAnalyticsServer(al *net.TCPListener, f io.Writer) error {
	for {
		c, err := al.AcceptTCP()
		if err != nil {
			return err
		}

		go handleAnalytics(c, f)
	}
}

type Analytic struct {
	Name, Command, IP string
	Time              time.Time
}

func (a *Analytic) WriteTo(f io.Writer) (int64, error) {
	n, err := fmt.Fprintf(f, "%s\t%q\t%s\t%s\n",
		a.Time.Format(time.DateTime), a.Command, a.Name, a.IP)

	return int64(n), err
}

func handleAnalytics(c *net.TCPConn, f io.Writer) {
	var sb strings.Builder

	if _, err := io.Copy(&sb, io.LimitReader(c, 4096)); err != nil {
		return
	}

	parts := strings.Split(sb.String(), "\x00")

	if len(parts) != 2 {
		return
	}

	ra := c.RemoteAddr().(*net.TCPAddr)

	a := Analytic{
		Name:    strings.TrimSpace(parts[0]),
		Command: strings.TrimSpace(parts[1]),
		IP:      ra.IP.String(),
		Time:    time.Now(),
	}

	_, err := a.WriteTo(f)
	if err != nil {
		slog.Error("error writing to output file", "err", err)
	}
}
