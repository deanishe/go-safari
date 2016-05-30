//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//

package safari

import "testing"

// TestNew asserts that Bookmarks.plist is found and read.
func TestNew(t *testing.T) {
	rb, err := New()
	if err != nil {
		t.Fatalf("Error reading Bookmarks.plist: %v", err)
	}
	if len(rb.Children) == 0 {
		t.Fatal("Root has 0 children")
	}
}

// TestNewParser asserts that Bookmarks.plist is found and read.
func TestNewParser(t *testing.T) {
	p, err := NewParser(BookmarksPath)
	if err != nil {
		t.Fatalf("Error reading Bookmarks.plist: %v", err)
	}
	if len(p.Raw.Children) == 0 {
		t.Fatal("Root has 0 children")
	}
}

// TestParserParse tests that Bookmarks and ReadingList are populated.
func TestParserParse(t *testing.T) {
	p, err := NewParser(BookmarksPath)
	if err != nil {
		t.Fatalf("Error reading Bookmarks.plist: %v", err)
	}
	if len(p.Bookmarks) == 0 {
		t.Fatal("Root has empty Bookmarks")
	}
	if len(p.BookmarksRL) == 0 {
		t.Fatal("Root has empty ReadingList")
	}
}

// TestParserFolders tests that folders are populated.
func TestParserFolders(t *testing.T) {
	p, err := NewParser(BookmarksPath)
	if err != nil {
		t.Fatalf("Error reading Bookmarks.plist: %v", err)
	}
	if len(p.Folders) == 0 {
		t.Fatal("Root has empty Folders")
	}
	if p.BookmarksBar == nil {
		t.Error("no BookmarksBar")
	}
	if p.BookmarksMenu == nil {
		t.Error("no BookmarksMenu")
	}
	if p.ReadingList == nil {
		t.Error("no ReadingList")
	}
}
