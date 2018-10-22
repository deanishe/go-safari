// Copyright (c) 2018 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

package cloud

import "testing"

func TestTabs(t *testing.T) {
	c, err := New(DefaultTabsPath)
	if err != nil {
		t.Fatal(err)
	}

	tabs, err := c.Tabs()
	if err != nil {
		t.Fatal(err)
	}

	if len(tabs) == 0 {
		t.Errorf("no cloud tabs found")
	}

	for _, tab := range tabs {
		if tab.Title == "" {
			t.Errorf("tab has no title: %#v", tab)
		}
		if tab.URL == "" {
			t.Errorf("tab has no URL: %#v", tab)
		}
		if tab.Device == "" {
			t.Errorf("tab has no device: %#v", tab)
		}
	}
}
