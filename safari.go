//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//

/*
Package safari provides access to Safari's windows, tabs, bookmarks etc.

Mac only.
*/
package safari

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kballard/go-osx-plist"
)

// Types of entries in Bookmarks.plist.
const (
	WebBookmarkTypeLeaf  = "WebBookmarkTypeLeaf"
	WebBookmarkTypeList  = "WebBookmarkTypeList"
	WebBookmarkTypeProxy = "WebBookmarkTypeProxy"
)

// Names of special folders.
const (
	NameBookmarksBar  = "BookmarksBar"
	NameBookmarksMenu = "BookmarksMenu"
	NameReadingList   = "com.apple.ReadingList"
)

var (
	// BookmarksPath is the path to Safari's exported bookmarks file on OS X
	BookmarksPath = filepath.Join(os.Getenv("HOME"), "Library/Safari/Bookmarks.plist")
	parsed        *Parser
)

// Folder is a folder of Bookmarks.
type Folder struct {
	Title           string
	Ancestors       []*Folder   // Last element is this Bookmark's parent
	Bookmarks       []*Bookmark // Bookmarks within this folder
	Folders         []*Folder   // Child folders
	isReadingList   bool
	isBookmarksBar  bool
	isBookmarksMenu bool
}

// IsReadingList returns true if this Folder is the user's Reading List.
func (f *Folder) IsReadingList() bool {
	return f.isReadingList
}

// IsBookmarksBar returns true if this Folder is the users's BookmarksBar.
func (f *Folder) IsBookmarksBar() bool {
	return f.isBookmarksBar
}

// IsBookmarksMenu returns true if this Folder is the users's BookmarksMenu.
func (f *Folder) IsBookmarksMenu() bool {
	return f.isBookmarksMenu
}

// Bookmark is a Safari bookmark.
type Bookmark struct {
	Title     string
	URL       string
	Ancestors []*Folder // Last element is this Bookmark's parent
	Preview   string
	UID       string
}

// Parser unmarshals a Bookmarks.plist.
type Parser struct {
	Raw           *RawBookmark // Bookmarks.plist data in "native" format
	Bookmarks     []*Bookmark  // Flat list of all bookmarks (excl. Reading List)
	BookmarksRL   []*Bookmark  // Flat list of all Reading List bookmarks
	Folders       []*Folder    // Flat list of all folders
	BookmarksBar  *Folder      // Folder for user's Bookmarks Bar
	BookmarksMenu *Folder      // Folder for user's Bookmarks Menu
	ReadingList   *Folder      // Folder for user's Reading List
}

// NewParser unmarshals a Bookmarks.plist file.
func NewParser(path string) (*Parser, error) {
	p := &Parser{}
	if err := p.Parse(path); err != nil {
		return nil, err
	}
	return p, nil
}

// Parse unmarshals a Bookmarks.plist.
func (p *Parser) Parse(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return p.parseData(data)
}

// parseData does the actual parsing.
func (p *Parser) parseData(data []byte) error {

	p.Raw = &RawBookmark{}
	p.Bookmarks = []*Bookmark{}
	p.BookmarksRL = []*Bookmark{}

	if _, err := plist.Unmarshal(data, p.Raw); err != nil {
		return err
	}

	if err := p.parse(p.Raw, []*Folder{}); err != nil {
		return err
	}

	return nil
}

// parse flattens the raw tree and parses the RawBookmarks into Bookmarks.
func (p *Parser) parse(root *RawBookmark, ancestors []*Folder) error {

	for _, rb := range root.Children {
		switch rb.Type {

		case WebBookmarkTypeProxy: // Ignore. Only History, which is empty
			continue
			// log.Printf("proxy=%s", rb.Title())

		case WebBookmarkTypeList: // Folder

			// var par *Folder
			// if len(parents) > 0 {
			// 	par = parents[len(parents)-1]
			// }
			f := &Folder{
				Title:     rb.Title(),
				Ancestors: ancestors,
			}

			// Add all folders to Parser
			p.Folders = append(p.Folders, f)

			if len(ancestors) == 0 { // Check if it's a special folder

				switch f.Title {

				case NameBookmarksBar:
					f.isBookmarksBar = true
					p.BookmarksBar = f

				case NameBookmarksMenu:
					f.isBookmarksMenu = true
					p.BookmarksMenu = f

				case NameReadingList:
					f.isReadingList = true
					p.ReadingList = f

				default:
					log.Printf("Unknown top-Level folder: %s", f.Title)
				}

			} else { // Just some normal folder
				par := ancestors[len(ancestors)-1]
				par.Folders = append(par.Folders, f)
			}

			if err := p.parse(rb, append(ancestors, f)); err != nil {
				return err
			}

		case WebBookmarkTypeLeaf: // Bookmark

			bm := &Bookmark{
				Title:     rb.Title(),
				URL:       rb.URL,
				Ancestors: ancestors,
				UID:       rb.UUID,
			}

			if rb.ReadingList != nil {
				bm.Preview = rb.ReadingList.PreviewText
			}

			par := ancestors[len(ancestors)-1]
			par.Bookmarks = append(par.Bookmarks, bm)

			if ancestors[0].isReadingList {
				// log.Printf("[ReadingList] + %s", bm.Title)
				p.BookmarksRL = append(p.BookmarksRL, bm)
			} else {
				// log.Printf("%v %s", parents, bm.Title)
				p.Bookmarks = append(p.Bookmarks, bm)
			}

		default:
			log.Printf("%v %s", ancestors, rb.Type)
		}
	}

	return nil
}

// RawRL contains the reading list metadata for a RawBookmark.
type RawRL struct {
	DateAdded       time.Time
	DateLastFetched time.Time
	DateLastViewed  time.Time
	PreviewText     string
}

// RawBookmark is the data model used in the Bookmarks.plist file.
type RawBookmark struct {
	RawTitle    string            `plist:"Title"`
	Type        string            `plist:"WebBookmarkType"`
	URL         string            `plist:"URLString"`
	UUID        string            `plist:"WebBookmarkUUID"`
	ReadingList *RawRL            `plist:"ReadingList"`
	URIDict     map[string]string `plist:"URIDictionary"`
	Children    []*RawBookmark
}

// Title returns either RawTitle (if set) or the title from URIDict.
func (rb *RawBookmark) Title() string {
	if rb.RawTitle != "" {
		return rb.RawTitle
	}
	return rb.URIDict["title"]
}

// New reads your Bookmarks.plist file and returns a tree of WebBookmarks.
func New() (*RawBookmark, error) {
	data, err := ioutil.ReadFile(BookmarksPath)
	if err != nil {
		return nil, err
	}
	rb := &RawBookmark{}
	if _, err := plist.Unmarshal(data, rb); err != nil {
		return nil, err
	}
	return rb, nil
}

func getParser() *Parser {
	if parsed != nil {
		return parsed
	}
	parsed, err := NewParser(BookmarksPath)
	if err != nil {
		panic(err)
	}
	return parsed
}

// Bookmarks returns all the user's bookmarks.
func Bookmarks() []*Bookmark {
	return getParser().Bookmarks
}

// BookmarksRL returns bookmarks for the user's Reading List.
func BookmarksRL() []*Bookmark {
	return getParser().BookmarksRL
}

// Folders returns all a user's bookmark folders.
func Folders() []*Folder {
	return getParser().Folders
}

// ReadingList returns user's Reading List folder.
func ReadingList() *Folder {
	return getParser().ReadingList
}

// BookmarksBar returns user's Bookmarks Bar folder.
func BookmarksBar() *Folder {
	return getParser().BookmarksBar
}

// BookmarksMenu returns user's Bookmarks Menu folder.
func BookmarksMenu() *Folder {
	return getParser().BookmarksMenu
}

// Filter calls Bookmarks() and returns the elements for which accept(bm) returns true.
func Filter(accept func(bm *Bookmark) bool) []*Bookmark {
	r := []*Bookmark{}

	for _, bm := range Bookmarks() {
		if accept(bm) {
			r = append(r, bm)
		}
	}

	return r
}
