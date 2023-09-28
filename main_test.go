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
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewAnalyticsServer(t *testing.T) {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{})
	if err != nil {
		t.Fatalf("unexpected error creating listener: %s", err)
	}

	var sb strings.Builder

	go newAnalyticsServer(l, &sb)

	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("unexpected error opening connection: %s", err)
	}

	const (
		user    = "USER"
		command = "/path/to/some/command"
	)

	_, err = io.WriteString(c, "USER\x00/path/to/some/command")
	if err != nil {
		t.Fatalf("unexpected error writing to connection: %s", err)
	}

	c.Close()

	time.Sleep(250 * time.Millisecond)

	expected := fmt.Sprintf("%q\t%s", command, user)

	if out := sb.String(); !strings.Contains(out, expected) {
		t.Errorf("%q did not contain %q", out, expected)
	}
}
