//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-22
//

package history

import (
	"net/url"
	"testing"
	"time"
)

func TestRecent(t *testing.T) {
	h, err := New(DefaultHistoryPath)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := h.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 10 {
		t.Errorf("bad no. of entries. Expected=10, Got=%d", len(entries))
	}

	for i, e := range entries {
		// fmt.Println(e.Title, e.Time)
		if e.Title == "" {
			t.Errorf("entry %d has no title: %#v", i, e)
		}
		if e.Time.After(time.Now()) {
			t.Errorf("entry %d in future: %#v", i, e)
		}
		u, err := url.Parse(e.URL)
		if err != nil {
			t.Errorf("entry %d has bad URL: %v", i, err)
			continue
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			t.Errorf("entry %d has bad scheme: %s", i, u.Scheme)
		}
	}
}

var testQueries = []string{"google", "alfred"}

func TestSearch(t *testing.T) {
	h, err := New(DefaultHistoryPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range testQueries {
		entries, err := h.Search(q)
		if err != nil {
			t.Errorf("search for '%s' failed: %v", q, err)
		}
		if len(entries) == 0 {
			t.Errorf("no results for '%s'", q)
		}
		if len(entries) > MaxSearchResults {
			t.Errorf("too many results for '%s': %d", q, len(entries))
		}
	}
}
