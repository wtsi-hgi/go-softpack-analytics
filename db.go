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
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB

	statements [3]*sql.Stmt
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	for _, table := range [...]string{
		`CREATE TABLE IF NOT EXISTS [events] (username TEXT, command string, ip string, time INTEGER)`,
		`CREATE TABLE IF NOT EXISTS [modules] (module TEXT PRIMARY KEY CHECK (module NOT LIKE ""), count INTEGER DEFAULT 1, firstuse INTEGER, lastuse INTEGER)`,
		`CREATE TABLE IF NOT EXISTS [usermodules] (module TEXT CHECK (module NOT LIKE ""), username TEXT, count INTEGER DEFAULT 1, firstuse INTEGER, lastuse INTEGER, CONSTRAINT usermodule UNIQUE(module, username))`,
		`CREATE INDEX IF NOT EXISTS modulename ON [usermodules] (module)`,
	} {
		if _, err := db.Exec(table); err != nil {
			return nil, fmt.Errorf("error creating initial table with sql %q: %w", table, err)
		}
	}

	d := &DB{db: db}

	for n, sql := range [...]string{
		"INSERT INTO [events] (username, command, ip, time) VALUES (?, ?, ?, ?);",
		"INSERT OR IGNORE INTO [modules] (module, firstuse, lastuse) VALUES (?, ?, ?) ON CONFLICT(module) DO UPDATE SET count = count + 1, firstuse = MIN(firstuse, excluded.firstuse), lastuse = MAX(lastuse, excluded.lastuse);",
		"INSERT OR IGNORE INTO [usermodules] (module, username, firstuse, lastuse) VALUES (?, ?, ?, ?) ON CONFLICT DO UPDATE SET count = count + 1, firstuse = MIN(firstuse, excluded.firstuse), lastuse = MAX(lastuse, excluded.lastuse);",
	} {
		if d.statements[n], err = db.Prepare(sql); err != nil {
			return nil, fmt.Errorf("error creating prepared statement with sql %q: %w", sql, err)
		}
	}

	return d, nil
}

func (d *DB) Add(username, command, module, ip string, now time.Time) error {
	secs := now.Unix()

	for n, params := range [...][]any{
		{username, command, ip, secs},
		{module, secs, secs},
		{module, username, secs, secs},
	} {
		if _, err := d.statements[n].Exec(params...); err != nil {
			return fmt.Errorf("error adding to database (%s, %s, %s, %d, %s): %w", module, username, ip, secs, command, err)
		}
	}

	return nil
}

func (d *DB) SaveTo(path string) error {
	_, err := d.db.Exec(fmt.Sprintf("VACUUM INTO %q", path))

	return err
}

func (d *DB) Close() error {
	return d.db.Close()
}
