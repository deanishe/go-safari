//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//
// TODO: Add iCloud devices & tabs as Folders and Bookmarks

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
	"strings"
	"time"

	"github.com/DHowett/go-plist"
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

// Types of objects with UIDs
const (
	TypeFolder   = "folder"
	TypeBookmark = "bookmark"
)

// Default options.
var (
	DefaultBookmarksPath      = filepath.Join(os.Getenv("HOME"), "Library/Safari/Bookmarks.plist")
	DefaultCloudTabsPath      = filepath.Join(os.Getenv("HOME"), "Library/SyncedPreferences/com.apple.Safari.plist")
	DefaultIgnoreBookmarklets = false

	parser *Parser // Default parser
)

// Item is a folder or bookmark.
type Item interface {
	Title() string
	UID() string
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

// Folder is a folder of Bookmarks.
type Folder struct {
	title           string
	Ancestors       []*Folder   // Last element is this Bookmark's parent
	Bookmarks       []*Bookmark // Bookmarks within this folder
	Folders         []*Folder   // Child folders
	uid             string
	isReadingList   bool
	isBookmarksBar  bool
	isBookmarksMenu bool
}

// Title returns Folder title and implements Item.
func (f *Folder) Title() string { return f.title }

// UID returns Folder title and implements Item.
func (f *Folder) UID() string { return f.uid }

// IsReadingList returns true if this Folder is the uidser's Reading List.
func (f *Folder) IsReadingList() bool { return f.isReadingList }

// IsBookmarksBar returns true if this Folder is the users's BookmarksBar.
func (f *Folder) IsBookmarksBar() bool { return f.isBookmarksBar }

// IsBookmarksMenu returns true if this Folder is the users's BookmarksMenu.
func (f *Folder) IsBookmarksMenu() bool { return f.isBookmarksMenu }

// Bookmark is a Safari bookmark.
type Bookmark struct {
	title     string
	URL       string
	Ancestors []*Folder // Last element is this Bookmark's parent
	Preview   string
	uid       string
}

// Title returns Bookmark title and implements Item.
func (bm *Bookmark) Title() string { return bm.title }

// UID returns Bookmark title and implements Item.
func (bm *Bookmark) UID() string { return bm.uid }

// Folder returns Bookmark's containing folder.
func (bm *Bookmark) Folder() *Folder {
	return bm.Ancestors[len(bm.Ancestors)-1]
}

// InReadingList returns true if Bookmark is from the Reading List.
func (bm *Bookmark) InReadingList() bool {
	return bm.Folder().IsReadingList()
}

// IsBookmarklet returns true if Bookmark is a bookmarklet.
func (bm *Bookmark) IsBookmarklet() bool {
	return strings.HasPrefix(bm.URL, "javascript:")
}

// Option sets a Parser option.
type Option func(*Parser)

// BookmarksPath sets the path to the Safari bookmarks plist.
func BookmarksPath(path string) Option {
	return func(p *Parser) { p.BookmarksPath = path }
}

// CloudTabsPath sets the path to the Safari iCloud tabs plist.
func CloudTabsPath(path string) Option {
	return func(p *Parser) { p.CloudTabsPath = path }
}

// IgnoreBookmarklets tells parser whether to ignore bookmarklets.
func IgnoreBookmarklets(ignore bool) Option {
	return func(p *Parser) { p.IgnoreBookmarklets = ignore }
}

// Parser unmarshals a Bookmarks.plist.
type Parser struct {
	BookmarksPath      string
	CloudTabsPath      string
	IgnoreBookmarklets bool         // Whether to ignore bookmarklets
	Raw                *RawBookmark // Bookmarks.plist data in "native" format
	Bookmarks          []*Bookmark  // Flat list of all bookmarks (excl. Reading List)
	BookmarksRL        []*Bookmark  // Flat list of all Reading List bookmarks
	Folders            []*Folder    // Flat list of all folders
	BookmarksBar       *Folder      // Folder for user's Bookmarks Bar
	BookmarksMenu      *Folder      // Folder for user's Bookmarks Menu
	ReadingList        *Folder      // Folder for user's Reading List
	uid2Folder         map[string]*Folder
	uid2Bookmark       map[string]*Bookmark
	uid2Type           map[string]string
}

// New unmarshals a Bookmarks.plist file.
func New(opts ...Option) (*Parser, error) {

	p := &Parser{
		BookmarksPath:      DefaultBookmarksPath,
		CloudTabsPath:      DefaultCloudTabsPath,
		IgnoreBookmarklets: DefaultIgnoreBookmarklets,
		uid2Folder:         map[string]*Folder{},
		uid2Bookmark:       map[string]*Bookmark{},
		uid2Type:           map[string]string{},
	}

	for _, opt := range opts {
		opt(p)
	}

	if err := p.Parse(); err != nil {
		return nil, err
	}

	return p, nil
}

// Configure applies an Option to Parser.
func (p *Parser) Configure(opts ...Option) {
	for _, opt := range opts {
		opt(p)
	}
}

// Parse unmarshals a Bookmarks.plist.
func (p *Parser) Parse() error {
	// TODO: Make Bookmarks.plist optional and add iCloud tabs
	data, err := ioutil.ReadFile(p.BookmarksPath)
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

	if err := p.parseRaw(p.Raw, []*Folder{}); err != nil {
		return err
	}

	return nil
}

// parse flattens the raw tree and parses the RawBookmarks into Bookmarks.
func (p *Parser) parseRaw(root *RawBookmark, ancestors []*Folder) error {

	for _, rb := range root.Children {
		switch rb.Type {

		case WebBookmarkTypeProxy: // Ignore. Only History, which is empty
			continue

		case WebBookmarkTypeList: // Folder

			f := &Folder{
				title:     rb.Title(),
				Ancestors: ancestors,
				uid:       rb.UUID,
			}

			// Add all folders to Parser
			p.Folders = append(p.Folders, f)
			p.uid2Folder[rb.UUID] = f
			p.uid2Type[rb.UUID] = TypeFolder

			if len(ancestors) == 0 { // Check if it's a special folder

				switch f.Title() {

				case NameBookmarksBar:
					f.title = "Favorites"
					f.isBookmarksBar = true
					p.BookmarksBar = f

				case NameBookmarksMenu:
					f.title = "Bookmarks Menu"
					f.isBookmarksMenu = true
					p.BookmarksMenu = f

				case NameReadingList:
					f.title = "Reading List"
					f.isReadingList = true
					p.ReadingList = f

				default:
					log.Printf("Unknown top-Level folder: %s", f.Title())
				}

			} else { // Just some normal folder
				par := ancestors[len(ancestors)-1]
				par.Folders = append(par.Folders, f)
			}

			if err := p.parseRaw(rb, append(ancestors, f)); err != nil {
				return err
			}

		case WebBookmarkTypeLeaf: // Bookmark

			if p.IgnoreBookmarklets && strings.HasPrefix(rb.URL, "javascript:") {
				continue
			}

			bm := &Bookmark{
				title:     rb.Title(),
				URL:       rb.URL,
				Ancestors: ancestors,
				uid:       rb.UUID,
			}

			p.uid2Bookmark[rb.UUID] = bm
			p.uid2Type[rb.UUID] = TypeBookmark

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

// BookmarkForUID returns Bookmark with given UID or nil.
func (p *Parser) BookmarkForUID(uid string) *Bookmark { return p.uid2Bookmark[uid] }

// FilterBookmarks returns all Bookmarks for which accept(bm) returns true.
func (p *Parser) FilterBookmarks(accept func(bm *Bookmark) bool) []*Bookmark {
	r := []*Bookmark{}

	for _, bm := range p.Bookmarks {
		if accept(bm) {
			r = append(r, bm)
		}
	}

	return r
}

// FindBookmark returns the first Bookmark for which accept(bm) returns true.
func (p *Parser) FindBookmark(accept func(bm *Bookmark) bool) *Bookmark {

	for _, bm := range p.Bookmarks {
		if accept(bm) {
			return bm
		}
	}
	return nil
}

// FilterFolders returns all Folders for which accept(bm) returns true.
func (p *Parser) FilterFolders(accept func(f *Folder) bool) []*Folder {
	r := []*Folder{}

	for _, f := range p.Folders {
		if accept(f) {
			r = append(r, f)
		}
	}

	return r
}

// FindFolder returns the first Folder for which accept(bm) returns true.
func (p *Parser) FindFolder(accept func(f *Folder) bool) *Folder {

	for _, f := range p.Folders {
		if accept(f) {
			return f
		}
	}
	return nil
}

// FolderForUID returns Folder with given UID or nil.
func (p *Parser) FolderForUID(uid string) *Folder { return p.uid2Folder[uid] }

// TypeForUID returns the type of item that UID refers to ("bookmark" or "folder").
func (p *Parser) TypeForUID(uid string) string { return p.uid2Type[uid] }

// getParser returns the default Parser, creating it if necessary.
func getParser() *Parser {
	if parser != nil {
		return parser
	}
	parser, err := New()
	if err != nil {
		panic(err)
	}
	return parser
}

// Configure sets options on the default parser.
func Configure(opts ...Option) { getParser().Configure(opts...) }

// Bookmarks returns all the user's bookmarks.
func Bookmarks() []*Bookmark { return getParser().Bookmarks }

// BookmarksRL returns bookmarks for the user's Reading List.
func BookmarksRL() []*Bookmark { return getParser().BookmarksRL }

// FilterBookmarks calls Bookmarks() and returns the elements for which accept(bm) returns true.
func FilterBookmarks(accept func(bm *Bookmark) bool) []*Bookmark {
	return getParser().FilterBookmarks(accept)
}

// FindBookmark returns the first Bookmark for which accept(bm) returns true.
// Returns nil if no match is found.
func FindBookmark(accept func(bm *Bookmark) bool) *Bookmark { return getParser().FindBookmark(accept) }

// BookmarkForUID returns Bookmark with UID uid or nil.
func BookmarkForUID(uid string) *Bookmark { return getParser().uid2Bookmark[uid] }

// BookmarksBar returns user's Bookmarks Bar folder.
func BookmarksBar() *Folder { return getParser().BookmarksBar }

// BookmarksMenu returns user's Bookmarks Menu folder.
func BookmarksMenu() *Folder { return getParser().BookmarksMenu }

// Folders returns all a user's bookmark folders.
func Folders() []*Folder { return getParser().Folders }

// ReadingList returns user's Reading List folder.
func ReadingList() *Folder { return getParser().ReadingList }

// FindFolder returns the first Folder for which accept(f) returns true.
// Returns nil if no match is found.
func FindFolder(accept func(f *Folder) bool) *Folder { return getParser().FindFolder(accept) }

// FilterFolders returns all Folders for which accept(f) returns true.
func FilterFolders(accept func(f *Folder) bool) []*Folder { return getParser().FilterFolders(accept) }

// FolderForUID returns Folder with UID uid or nil.
func FolderForUID(uid string) *Folder { return getParser().uid2Folder[uid] }
