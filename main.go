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
	"compress/gzip"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	port := flag.Uint64("p", 1234, "port to listen on for analytics")
	output := flag.String("d", "", "db file")
	tsv := flag.String("t", "", "import file")
	flag.Parse()

	if *tsv != "" {
		if err := importAndSaveData(*tsv, *output); err != nil {
			return err
		}
	}

	al, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(*port)})
	if err != nil {
		return err
	}

	db, err := NewDB(*output)
	if err != nil {
		return fmt.Errorf("error opening database (%s): %w", *output, err)
	}

	slog.Info("Server Started…")
	defer slog.Info("…Server Stopped")

	return newAnalyticsServer(al, db)
}

func newAnalyticsServer(al *net.TCPListener, db *DB) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	go func() {
		sig := make(chan os.Signal, 1)

		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		<-sig

		al.Close()
	}()

	for {
		c, err := al.AcceptTCP()
		if err != nil {
			return err
		}

		wg.Add(1)

		go handleAnalytics(c, db, &wg)
	}
}

func handleAnalytics(c *net.TCPConn, db *DB, wg *sync.WaitGroup) {
	defer wg.Done()

	var sb strings.Builder

	if _, err := io.Copy(&sb, io.LimitReader(c, 4096)); err != nil {
		return
	}

	parts := strings.Split(sb.String(), "\x00")

	if len(parts) != 2 {
		return
	}

	username := strings.TrimSpace(parts[0])
	command := strings.TrimSpace(parts[1])
	ip := c.RemoteAddr().(*net.TCPAddr).IP

	if err := db.Add(username, command, moduleFromCommand(command), ip.String(), time.Now()); err != nil {
		slog.Error("error writing to database", "err", err)
	}
}

func moduleFromCommand(command string) string {
	module := filepath.Dir(command)
	module = strings.TrimPrefix(module, "/software/hgi/softpack/installs/")
	module = strings.TrimPrefix(module, "/software/hgi/installs/")
	module = strings.TrimSuffix(module, "-scripts")

	if module == "." || strings.HasPrefix(module, "conda-audited") || strings.HasPrefix(module, "micromamba") || strings.HasPrefix(module, "/nfs/users/nfs_s/sb10/src/hgi/conda-audited") {
		return ""
	}

	return module
}

func importAndSaveData(tsv, output string) error {
	db, err := NewDB(":memory:")
	if err != nil {
		return fmt.Errorf("error opening in-memory db: %w", err)
	}

	slog.Info("Importing…")
	if err := importData(db, tsv); err != nil {
		return fmt.Errorf("error importing data: %w", err)
	}

	slog.Info("Exporting…")

	if err := db.SaveTo(output); err != nil {
		return fmt.Errorf("error exporting in-memory db to disk: %w", err)
	}

	slog.Info("…Done")

	return nil
}

func importData(db *DB, path string) error {
	var r io.Reader

	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		defer f.Close()

		if strings.HasSuffix(path, ".gz") {
			if r, err = gzip.NewReader(f); err != nil {
				return fmt.Errorf("failed to read gzip compressed input: %w", err)
			}
		} else {
			r = f
		}
	}

	return readCSVIntoDB(csv.NewReader(r), db)
}

func readCSVIntoDB(reader *csv.Reader, db *DB) error {
	reader.Comma = '\t'
	count := 0

	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		} else if len(row) < 4 {
			continue
		}

		date, command, user, ip := row[0], row[1], row[2], row[3]

		d, err := time.Parse(time.DateTime, date)
		if err != nil {
			continue
		} else if err = db.Add(user, command, moduleFromCommand(command), ip, d); err != nil {
			return fmt.Errorf("error adding to database: %w", err)
		}

		count++

		if count%1000 == 0 {
			fmt.Printf("\r%d", count)
		}
	}
}
