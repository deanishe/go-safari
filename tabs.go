//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//

package safari

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/juju/deputy"
)

// Safari automation scripts
var (
	// jsGetTabs -> JSON
	jsGetTabs = `

    ObjC.import('stdlib')
    ObjC.import('stdio')

    function getWindows() {

      var safari = Application('Safari')
      safari.includeStandardAdditions = true

      var results = []
      var wins = safari.windows

      for (i=0; i<wins.length; i++) {
        var data = {'index': i+1, 'tabs': []},
          w = wins[i],
          tabs = w.tabs

        // Ignore non-browser windows
        try {
          data['activeTab'] = w.currentTab().index()
        }
        catch (e) {
          console.log('Ignoring window ' + (i+1))
          continue
        }

        // Tabs
        for (j=0; j<tabs.length; j++) {
          var t = tabs[j]
          data.tabs.push({
            'title': t.name(),
            'url': t.url(),
            'index': j+1,
            'windowIndex': i+1
          })
        }

        results.push(data)
      }
      return results
    }

    function run(argv) {
      return JSON.stringify(getWindows())
    }
    `

	// jsActivate <window-number> [<tab-number>] -> nil
	jsActivate = `

    ObjC.import('stdlib')

    function activateTab(winIdx, tabIdx) {
      var safari = Application('Safari')
      safari.includeStandardAdditions = true

      try {
        var win = safari.windows[winIdx-1]()
      }
      catch (e) {
        console.log('Invalid window: ' + winIdx)
        $.exit(1)
      }

      if (tabIdx == 0) { // Activate window
        safari.activate()
        win.visible = false
        win.visible = true

        return
      }

      // Find tab to activate
      try {
        var tab = win.tabs[tabIdx-1]()
      }
      catch (e) {
        console.log('Invalid tab for window ' + winIdx + ': ' + tabIdx)
        $.exit(1)
      }

      // Activate window and tab if it's not the current tab
      safari.activate()
      win.visible = false
      win.visible = true

      if (!tab.visible()) {
        win.currentTab = tab
      }

    }

    function run(argv) {
      var win = 0,
        tab = 0;

      win = parseInt(argv[0], 10)
      if (argv.length > 1) {
        tab = parseInt(argv[1], 10)
      }

      if (isNaN(win)) {
        console.log('Invalid window: ' + win)
        $.exit(1)
      }

      if (isNaN(tab)) {
        console.log('Invalid tab: ' + tab)
        $.exit(1)
      }

      activateTab(win, tab)
    }
    `

	jsClose = `
    Array.prototype.contains = function(o) {
      return this.indexOf(o) > -1
    }

    ObjC.import('stdlib');

    // Permissible targets
    var whats = ['win', 'tab', 'tabs-other', 'tabs-left', 'tabs-right'],
      app = Application('Safari');
      app.includeStandardAdditions = true;


    // usage | Print help to STDOUT
    function usage() {

      console.log('SafariClose.js (win|tab|tabs-other|tabs-left|tabs-right) [<win>] [<tab>]');
      console.log('');
      console.log('Close specified window and/or tab(s). If not specified, <win> and <tab>');
      console.log('default to the frontmost window and current tab respectively.');
      console.log('');
      console.log('Usage:');
      console.log('    SafariClose.js win [<win>]');
      console.log('    SafariClose.js (tab|tabs-other|tabs-left|tabs-right) [<win>] [<tab>]');
      console.log('    SafariClose.js -h');
      console.log('');
      console.log('Options:');
      console.log('    -h    Show this help message and exit.');

    }


    // closeWindow | Close the specified Safari window
    function closeWindow(winIdx) {
      var win = app.windows[winIdx-1];
      win.close();
    }

    // getCurrentTab | Return the index of the current tab of frontmost window
    function getCurrentTab() {
      return app.windows[0].currentTab.index()
    }

    // closeTabs | tabFunc(idx, tab) is called for each tab in the window.
    // Tab is closed if it returns true.
    function closeTabs(winIdx, tabFunc) {

      var win = app.windows[winIdx-1],
          tabs = win.tabs,
          current = win.currentTab,
          toClose = [];

      // Loop backwards, so tab indices don't change as we close them
      for (i=tabs.length-1; i>-1; i--) {
        var tab = tabs[i];
        if (tabFunc(i+1, tab)) {
          console.log('Closing tab ' + (i+1) + ' ...');
          tab.close();
        }
      }

    }


    function run(argv) {
      var what = argv[0],
          winIdx = 1,  // default to frontmost window
          tabIdx = 0;

      if (argv.contains('-h') || argv.contains('--help')) {
        usage();
        $.exit(0);
      }

      // Validate arguments
      if (!whats.contains(what)) {
        console.log('Invalid target: ' + what);
        console.log('');
        usage();
        $.exit(1);
      }

      if (typeof(argv[1]) != 'undefined') {
        winIdx = parseInt(argv[1], 10);
        if (isNaN(winIdx)) {
          console.log('Invalid window number: ' + argv[1]);
          $.exit(1);
        }
      }

      if (what != 'win') {
        if (typeof(argv[2]) != 'undefined') {
          var tabIdx = parseInt(argv[2], 10);
          if (isNaN(tabIdx)) {
            console.log('Invalid tab number for window ' + winIdx + ': ' + argv[2]);
            $.exit(1);
          }
        } else {
          tabIdx = getCurrentTab();
        }
      }

      console.log('winIdx=' + winIdx + ', tabIdx=' + tabIdx);

      // Let's close some shit
      if (what == 'win') {

        return closeWindow(winIdx)

      } else if (what == 'tab') {

        //return closeTab(winIdx, tabIdx)
        return closeTabs(winIdx, function(i, t) {
          return i === tabIdx
        })

      } else if (what == 'tabs-other') {

        return closeTabs(winIdx, function(i, t) {
          return i != tabIdx
        })

      } else if (what == 'tabs-left') {

        return closeTabs(winIdx, function(i, t) {
          return i < tabIdx
        })


      } else if (what == 'tabs-right') {

        return closeTabs(winIdx, function(i, t) {
          return i > tabIdx
        })

      }
    }
    `
)

// Tab is a Safari tab.
type Tab struct {
	Index       int
	WindowIndex int
	Title       string
	URL         string
}

// Window is a Safari window.
type Window struct {
	Index     int
	ActiveTab int
	Tabs      []*Tab
}

// Windows returns information about Safari's open windows.
//
// NOTE: This function takes a long time (~0.5 seconds) to complete as
// it calls Safari via the Scripting Bridge, which is slow as shit.
//
// You would be wise to cache these data for a few seconds.
func Windows() ([]*Window, error) {
	wins := []*Window{}

	if err := runJXA2JSON(jsGetTabs, &wins); err != nil {
		return nil, err
	}

	return wins, nil
}

// Activate activates the specified Safari window (and tab). If tab is 0,
// the active tab will not be changed.
func Activate(win, tab int) error {

	args := []string{fmt.Sprintf("%d", win)}
	if tab > 0 {
		args = append(args, fmt.Sprintf("%d", tab))
	}

	if _, err := runJXA(jsActivate, args...); err != nil {
		return err
	}
	return nil
}

// ActivateTab activates the specified tab.
func ActivateTab(win, tab int) error {
	return Activate(win, tab)
}

// ActivateWin activates the specified window.
func ActivateWin(win int) error {
	return Activate(win, 0)
}

// closeStuff runs script jsClose with the given arguments.
func closeStuff(what string, win, tab int) error {

	if win == 0 { // Default to frontmost window
		win = 1
	}
	args := []string{what, fmt.Sprintf("%d", win)}

	if tab > 0 {
		args = append(args, fmt.Sprintf("%d", tab))
	}

	if _, err := runJXA(jsClose, args...); err != nil {
		return err
	}

	return nil
}

// Close closes the specified tab.
// If win is 0, the frontmost window is assumed. If tab is 0, current tab is
// assumed.
func Close(win, tab int) error { return closeStuff("tab", win, tab) }

// CloseWin closes the specified window. If win is 0, the frontmost window is closed.
func CloseWin(win int) error { return closeStuff("win", win, 0) }

// CloseTab closes the specified tab. If win is 0, frontmost window is assumed.
// If tab is 0, current tab is closed.
func CloseTab(win, tab int) error { return closeStuff("tab", win, tab) }

// CloseTabsOther closes all other tabs in win.
func CloseTabsOther(win, tab int) error { return closeStuff("tabs-other", win, tab) }

// CloseTabsLeft closes tabs to the left of the specified one.
func CloseTabsLeft(win, tab int) error { return closeStuff("tabs-left", win, tab) }

// CloseTabsRight closes tabs to the right of the specified one.
func CloseTabsRight(win, tab int) error { return closeStuff("tabs-right", win, tab) }

// runJXA executes JavaScript script with /usr/bin/osascript and returns the
// script's output on STDOUT.
func runJXA(script string, argv ...string) ([]byte, error) {

	data := []byte{}

	d := deputy.Deputy{
		Errors:    deputy.FromStderr,
		StdoutLog: func(b []byte) { data = append(data, b...) },
		Timeout:   time.Second * 5,
	}

	cmd := "/usr/bin/osascript"
	args := []string{"-l", "JavaScript", "-e", script}

	if len(argv) > 0 {
		args = append(args, argv...)
	}

	if err := d.Run(exec.Command(cmd, args...)); err != nil {
		return data, err
	}

	return data, nil
}

// runJXA2JSON executes a JXA script and unmarshals its output to target using
// json.Unmarshal()
func runJXA2JSON(script string, target interface{}, argv ...string) error {
	data, err := runJXA(script, argv...)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return err
	}

	return nil
}
