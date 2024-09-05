/*******************************************************************************
 * Copyright (c) 2024 Genome Research Ltd.
 *
 * Authors:
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
	"net"
	"strings"
	"testing"
	"time"
)

func TestDB(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error creating DB: %s", err)
	}

	for n, test := range [...]struct {
		Username            string
		Command             string
		Module              string
		IP                  net.IP
		Time                time.Time
		ExpectedEvents      string
		ExpectedModules     string
		ExpectedUserModules string
	}{
		{
			"userA",
			"some command 1",
			"",
			net.IPv4(192, 168, 1, 1),
			time.Unix(1, 0),
			"userA,some command 1,192.168.1.1,1\n",
			"",
			"",
		},
		{
			"userA",
			"some command 2",
			"moduleA",
			net.IPv4(192, 168, 1, 1),
			time.Unix(2, 0),
			"userA,some command 1,192.168.1.1,1\n" +
				"userA,some command 2,192.168.1.1,2\n",
			"moduleA,1,2,2\n",
			"moduleA,userA,1,2,2\n",
		},
		{
			"userA",
			"some command 3",
			"moduleA",
			net.IPv4(192, 168, 1, 1),
			time.Unix(3, 0),
			"userA,some command 1,192.168.1.1,1\n" +
				"userA,some command 2,192.168.1.1,2\n" +
				"userA,some command 3,192.168.1.1,3\n",
			"moduleA,2,2,3\n",
			"moduleA,userA,2,2,3\n",
		},
		{
			"userB",
			"some command 2",
			"moduleA",
			net.IPv4(192, 168, 1, 2),
			time.Unix(4, 0),
			"userA,some command 1,192.168.1.1,1\n" +
				"userA,some command 2,192.168.1.1,2\n" +
				"userA,some command 3,192.168.1.1,3\n" +
				"userB,some command 2,192.168.1.2,4\n",
			"moduleA,3,2,4\n",
			"moduleA,userA,2,2,3\n" +
				"moduleA,userB,1,4,4\n",
		},
		{
			"userB",
			"some command 4",
			"moduleB",
			net.IPv4(192, 168, 1, 2),
			time.Unix(5, 0),
			"userA,some command 1,192.168.1.1,1\n" +
				"userA,some command 2,192.168.1.1,2\n" +
				"userA,some command 3,192.168.1.1,3\n" +
				"userB,some command 2,192.168.1.2,4\n" +
				"userB,some command 4,192.168.1.2,5\n",
			"moduleA,3,2,4\n" +
				"moduleB,1,5,5\n",
			"moduleA,userA,2,2,3\n" +
				"moduleA,userB,1,4,4\n" +
				"moduleB,userB,1,5,5\n",
		},
		{
			"userB",
			"some command 4",
			"moduleB",
			net.IPv4(192, 168, 1, 2),
			time.Unix(1, 0),
			"userA,some command 1,192.168.1.1,1\n" +
				"userA,some command 2,192.168.1.1,2\n" +
				"userA,some command 3,192.168.1.1,3\n" +
				"userB,some command 2,192.168.1.2,4\n" +
				"userB,some command 4,192.168.1.2,5\n" +
				"userB,some command 4,192.168.1.2,1\n",
			"moduleA,3,2,4\n" +
				"moduleB,2,1,5\n",
			"moduleA,userA,2,2,3\n" +
				"moduleA,userB,1,4,4\n" +
				"moduleB,userB,2,1,5\n",
		},
	} {
		if err := db.Add(test.Username, test.Command, test.Module, test.IP.String(), test.Time); err != nil {
			t.Errorf("test %d: unexpected error adding event: %s", n+1, err)

			continue
		}

		if evtable := dumpTable(t, db, "events"); evtable != test.ExpectedEvents {
			t.Errorf("test %d: expected events table to be:\n%s\ngot:\n%s", n+1, test.ExpectedEvents, evtable)
		} else if mdtable := dumpTable(t, db, "modules"); mdtable != test.ExpectedModules {
			t.Errorf("test %d: expected modules table to be:\n%s\ngot:\n%s", n+1, test.ExpectedModules, mdtable)
		} else if umtable := dumpTable(t, db, "usermodules"); umtable != test.ExpectedUserModules {
			t.Errorf("test %d: expected usermodules table to be:\n%s\ngot:\n%s", n+1, test.ExpectedUserModules, umtable)
		}
	}
}

func dumpTable(t *testing.T, db *DB, table string) string {
	t.Helper()

	rows, err := db.db.Query("SELECT * FROM [" + table + "]")
	if err != nil {
		t.Fatalf("unexpected error querying table (%s): %s", table, err)
	}

	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("unexpected error querying table (%s): %s", table, err)
	}

	results := make([]any, len(cols))
	ptrResults := make([]any, len(cols))

	for n := range results {
		ptrResults[n] = &results[n]
	}

	format := "%v" + strings.Repeat(",%v", len(cols)-1) + "\n"

	var sb strings.Builder

	for rows.Next() {
		if err := rows.Scan(ptrResults...); err != nil {
			t.Fatalf("error scanning row data: %s", err)
		}

		fmt.Fprintf(&sb, format, results...)
	}

	return sb.String()
}
