//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-22
//

// Package history provides access to Safari's history.
//
// As the exported history was removed in High Sierra, this package
// accesses Safari's private sqlite3 database.
package history

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	// sqlite3 registers itself with sql
	_ "github.com/mattn/go-sqlite3"
)

var (
	// DefaultHistoryPath is where Safari's history database is stored.
	DefaultHistoryPath = filepath.Join(os.Getenv("HOME"), "Library/Safari/History.db")
	// MaxSearchResults is the number of results to return from a search.
	MaxSearchResults = 200
	history          *History
	// NSDate epoch starts at 00:00:00 on 1/1/2001 UTC
	tsOffset = 978307200.0
)

func init() {
	var err error
	history, err = New(DefaultHistoryPath)
	if err != nil {
		panic(err)
	}
}

// Entry is a History entry.
type Entry struct {
	Title string
	URL   string
	Time  time.Time
}

// History is a Safari history.
type History struct {
	DB *sql.DB
}

// New creates a new History from a Safari history database.
func New(filename string) (*History, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	return &History{db}, nil
}

// Recent returns the specified number of most recent items from History.
func Recent(count int) ([]*Entry, error) { return history.Recent(count) }
func (h *History) Recent(count int) ([]*Entry, error) {
	q := `
	SELECT url, visit_time, title
		FROM history_items
			LEFT JOIN history_visits
				ON history_visits.history_item = history_items.id
		WHERE title <> '' AND url LIKE 'http%'
		ORDER BY visit_time DESC LIMIT ?`

	return h.query(q, count)
}

// Search searches all History entries.
func Search(query string) ([]*Entry, error) { return history.Search(query) }
func (h *History) Search(query string) ([]*Entry, error) {
	query = "%" + query + "%"
	q := `
	SELECT url, visit_time, title
		FROM history_items
			LEFT JOIN history_visits
				ON history_visits.history_item = history_items.id
		WHERE title <> '' AND title LIKE ? AND url LIKE 'http%'
		ORDER BY visit_time DESC LIMIT ?`

	return h.query(q, query, MaxSearchResults)
}

// query runs an SQL query against the database.
func (h *History) query(q string, args ...interface{}) ([]*Entry, error) {
	var (
		url, title string
		when       float64
		ts         int64
		t          time.Time
		entries    []*Entry
	)
	rows, err := h.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		rows.Scan(&url, &when, &title)
		ts = int64(when + tsOffset)
		t = time.Unix(ts, 0).Local()
		entries = append(entries, &Entry{title, url, t})
	}

	return entries, nil
}
