//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//

// Command safari is an Alfred 3 workflow for manipulating Safari and searching its data.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"gopkg.in/alecthomas/kingpin.v2"

	"gogs.deanishe.net/deanishe/go-safari"
)

const (
	// Version is the version number of the program
	Version = "0.1.0"
)

var (
	startTime time.Time

	// CLI arguments
	outputJSON           bool
	colourisedOutput     bool
	targetWin, targetTab int
	listContentType      string
	closeTargetType      string

	// Kingpin components
	app                            *kingpin.Application
	activateCmd, listCmd, closeCmd *kingpin.CmdClause

	// Colours
	yellow  = color.New(color.FgYellow)
	white   = color.New(color.FgWhite)
	magenta = color.New(color.FgMagenta)
	blue    = color.New(color.FgBlue)
	cyan    = color.New(color.FgCyan)

	// tabsCmd         = app.Command("tabs", "List open tabs.")
	// bookmarksCmd    = app.Command("bookmarks", "List bookmarks.")
	// foldersCmd      = app.Command("folders", "List bookmark folders.")

)

func init() {

	startTime = time.Now()
	log.SetFlags(0)

	// Set up Kingpin
	app = kingpin.New("safari", "Manipulate Safari and search its data.")
	app.HelpFlag.Short('h')
	app.Version(Version)

	// Global flags
	app.Flag("colour", "Turn colourised output on/off (default=on).").Default("true").BoolVar(&colourisedOutput)

	// Activate
	activateCmd = app.Command("activate", "Active a Safari window or tab.").Alias("a")
	activateCmd.Arg("window", "The window to activate.").Required().IntVar(&targetWin)
	activateCmd.Arg("tab", "The tab to activate.").IntVar(&targetTab)

	// List
	listCmd = app.Command("list", "List Safari bookmarks, folders or tabs.").Alias("l")
	listCmd.Flag("json", "Output JSON, not text.").Short('j').BoolVar(&outputJSON)
	listCmd.Arg("type", "Type of data to list (bookmarks, folders, readlist or tabs).").
		Required().
		EnumVar(&listContentType, "b", "bookmarks", "f", "folders", "r", "readlist", "t", "tabs")

	// Close
	closeCmd = app.Command("close", "Close Safari windows and/or tabs.").Alias("c")
	closeCmd.Arg("what", "What to close (win, tab, tabs-other, tabs-left or tabs-right).").
		Required().
		EnumVar(&closeTargetType, "w", "win", "t", "tab", "to", "tabs-other",
			"tl", "tabs-left", "tr", "tabs-right")
	closeCmd.Arg("window", "The target window.").Default("1").IntVar(&targetWin)
	closeCmd.Arg("tab", "The target tab.").IntVar(&targetTab)
}

// node is for pretty-printing trees of colourful strings.
type node struct {
	name     string
	children []*node
	last     bool
	colour   *color.Color
}

// prettyPrint prints a lovely tree to STDOUT.
func (n *node) prettyPrint(indent string, last bool, root bool) {
	fmt.Print(indent)
	if root {
		indent += ""
	} else if last {
		fmt.Print("└─ ")
		indent += "   "
	} else {
		fmt.Print("├─ ")
		indent += "│  "
	}
	// fmt.Println(n.name)
	n.colour.Println(n.name)

	for i, c := range n.children {
		isLast := (i == len(n.children)-1)
		c.prettyPrint(indent, isLast, false)
	}
}

// nodeTree builds a tree of nodes for Folder f.
func nodeTree(f *safari.Folder, includeBookmarks bool) *node {
	n := &node{
		name:   f.Title + "/",
		last:   true,
		colour: yellow,
	}

	for i, f2 := range f.Folders {
		n2 := nodeTree(f2, includeBookmarks)
		n2.last = (i == len(f.Folders)-1)
		n.children = append(n.children, n2)

	}

	if includeBookmarks {
		for i, bm := range f.Bookmarks {
			// Ignore bookmarklets
			if strings.HasPrefix(bm.URL, "javascript:") {
				continue
			}
			n2 := &node{name: bm.Title, colour: blue}
			n2.last = (i == len(f.Bookmarks)-1)
			n.children = append(n.children, n2)
		}
	}

	return n
}

// printFolder prints Folder f to STDOUT as a tree.
func printFolder(f *safari.Folder, includeBookmarks bool) {
	n := nodeTree(f, includeBookmarks)
	n.prettyPrint("", true, true)
}

// printWindows prints windows and tabs to STDOUT as a tree.
func printWindows(wins []*safari.Window) {
	n := &node{
		name:   "Windows",
		last:   true,
		colour: yellow,
	}

	for i, w := range wins {

		n2 := &node{
			name:   fmt.Sprintf("Window %d", w.Index),
			colour: blue,
		}
		if i == len(wins)-1 {
			n2.last = true
		}

		for j, t := range w.Tabs {
			c := blue
			var active string
			if t.Index == w.ActiveTab {
				c = cyan
				if color.NoColor {
					active = "* "
				}
			}
			name := fmt.Sprintf("[%2d] %s%s", t.Index, active, t.Title)

			n3 := &node{
				name:   name,
				colour: c,
				last:   (j == len(w.Tabs)-1),
			}

			n2.children = append(n2.children, n3)
		}

		n.children = append(n.children, n2)
	}

	n.prettyPrint("", true, true)
}

func printJSON(o interface{}) error {
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

// doListTabs prints a tree of Safari windows and tabs to STDOUT.
func doListTabs() error {

	wins, err := safari.Windows()
	if err != nil {
		return fmt.Errorf("Error communicating with Safari: %v", err)
	}

	if outputJSON {
		return printJSON(wins)
	}

	printWindows(wins)
	return nil
}

// jsonBookmark is a wrapper for safari.Bookmark that eliminates the circular
// references.
type jsonBookmark struct {
	Title     string
	URL       string
	Ancestors []string
	Preview   string
	UID       string
}

// newJSONBookmark populates an jsonBookmark based on a safari.Bookmark.
func newJSONBookmark(bm *safari.Bookmark) *jsonBookmark {
	var ancestors []string
	for _, f := range bm.Ancestors {
		ancestors = append(ancestors, f.Title)
	}
	return &jsonBookmark{
		Title:     bm.Title,
		URL:       bm.URL,
		Ancestors: ancestors,
		Preview:   bm.Preview,
		UID:       bm.UID}

}

// jsonFolder is a wrapper for safari.Folder that removes circular references.
type jsonFolder struct {
	Title     string
	Ancestors []string
	Bookmarks []*jsonBookmark
}

// newJSONFolder populates a jsonFolder from a safari.Folder.
func newJSONFolder(f *safari.Folder) *jsonFolder {
	var ancestors []string
	for _, f2 := range f.Ancestors {
		ancestors = append(ancestors, f2.Title)
	}
	jf := &jsonFolder{
		Title:     f.Title,
		Ancestors: ancestors,
		Bookmarks: []*jsonBookmark{},
	}
	for _, bm := range f.Bookmarks {
		if strings.HasPrefix(bm.URL, "javascript:") {
			continue
		}
		jf.Bookmarks = append(jf.Bookmarks, newJSONBookmark(bm))
	}
	return jf
}

// doListTabs prints a tree of Safari's Bookmarks Bar to STDOUT.
func doListBookmarks() error {

	p, err := safari.New(nil)
	if err != nil {
		return err
	}

	if outputJSON {
		output := []*jsonBookmark{}
		for _, bm := range p.Bookmarks {
			if strings.HasPrefix(bm.URL, "javascript:") {
				continue
			}
			output = append(output, newJSONBookmark(bm))
		}

		return printJSON(output)
	}

	printFolder(p.BookmarksBar, true)
	return nil
}

func flattenFolderTree(f *safari.Folder) []*safari.Folder {
	r := []*safari.Folder{f}

	for _, f2 := range f.Folders {
		r = append(r, flattenFolderTree(f2)...)
	}

	return r
}

// doListTabs prints a tree of the folders within Safari's Bookmarks Bar to STDOUT.
func doListFolders() error {

	p, err := safari.New(nil)
	if err != nil {
		return err
	}

	if outputJSON {
		output := []*jsonFolder{}

		for _, f := range flattenFolderTree(p.BookmarksBar) {
			output = append(output, newJSONFolder(f))
		}

		return printJSON(output)
	}

	printFolder(p.BookmarksBar, false)

	return nil
}

// doListReadingList prints the titles of the items in Safari's Reading List.
func doListReadingList() error {

	p, err := safari.New(nil)
	if err != nil {
		return err
	}

	f := p.ReadingList

	if outputJSON {
		output := []*jsonBookmark{}
		for _, bm := range f.Bookmarks {
			if strings.HasPrefix(bm.URL, "javascript:") {
				continue
			}
			output = append(output, newJSONBookmark(bm))
		}

		return printJSON(output)
	}

	fStr := fmt.Sprintf("[%%%dd] %%s\n", len(fmt.Sprintf("%d", len(f.Bookmarks))))

	for i, bm := range f.Bookmarks {
		fmt.Printf(fStr, i+1, bm.Title)
	}
	// printFolder(p.BookmarksBar, false)

	return nil
}

// doActivate activates the specified window/tab.
func doActivate() error {
	log.Printf("Activating %vx%v", targetWin, targetTab)
	return safari.Activate(targetWin, targetTab)
}

// doList calls other doListXYZ() commands based on command-line args.
func doList() error {

	switch listContentType {

	case "b", "bookmarks":
		return doListBookmarks()

	case "f", "folders":
		return doListFolders()

	case "r", "readlist":
		return doListReadingList()

	case "t", "tabs":
		return doListTabs()
	}

	return nil
}

// doClose closes the specified window/tab(s).
func doClose() error {

	log.Printf("target=%s, win=%d, tab=%d", closeTargetType, targetWin, targetTab)

	switch closeTargetType {

	case "w", "win":
		return safari.CloseWin(targetWin)

	case "t", "tab":
		return safari.CloseTab(targetWin, targetTab)

	case "to", "tabs-other":
		return safari.CloseTabsOther(targetWin, targetTab)

	case "tl", "tabs-left":
		return safari.CloseTabsLeft(targetWin, targetTab)

	case "tr", "tabs-right":
		return safari.CloseTabsRight(targetWin, targetTab)

	default:
		return fmt.Errorf("Unknown target: %s", closeTargetType)

	}

}

func main() {

	var err error

	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if !colourisedOutput {
		color.NoColor = true
	}

	switch cmd {

	// case bookmarksCmd.FullCommand():
	// 	err = doListBookmarks()
	//
	// case foldersCmd.FullCommand():
	// 	err = doListFolders()
	//
	// case tabsCmd.FullCommand():
	// 	err = doListTabs()

	case activateCmd.FullCommand():
		err = doActivate()
		app.FatalIfError(err, "%s", "Safari command failed")

	case closeCmd.FullCommand():
		err = doClose()
		app.FatalIfError(err, "%s", "Safari command failed")

	case listCmd.FullCommand():
		err = doList()

	default:
		fmt.Printf("json=%v", outputJSON)
	}

	app.FatalIfError(err, "%s", "")

	log.Printf("--------- %v ---------", time.Since(startTime))

}
