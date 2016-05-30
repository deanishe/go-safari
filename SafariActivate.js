#!/usr/bin/env osascript -l JavaScript
//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-30
//

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
