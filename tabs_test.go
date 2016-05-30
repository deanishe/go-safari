//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//

package safari

import "testing"

// TestWindows tests that Safari windows and tabs are correctly read.
func TestWindows(t *testing.T) {
	wins, err := Windows()
	if err != nil {
		t.Fatalf("Error getting Safari windows: %v", err)
	}
	if len(wins) == 0 {
		t.Fatal("No windows.")
	}

	for _, w := range wins {
		if w.Index == 0 {
			t.Error("Window index is 0")
		}
		if len(w.Tabs) == 0 {
			t.Errorf("No tabs in window %d", w.Index)
		}
		if w.ActiveTab == 0 {
			t.Errorf("No active tab in window %d", w.Index)
		}
		for _, tab := range w.Tabs {
			if tab.Index == 0 {
				t.Error("Tab index is 0")
			}
			if tab.Title == "" {
				t.Error("Tab has no title")
			}
			if tab.URL == "" {
				t.Error("Tab has no URL")
			}
			if tab.WindowIndex != w.Index {
				t.Errorf("WindowIndex != w.Index. Expected=%v, Got=%v", w.Index, tab.WindowIndex)
			}
		}
	}

}
