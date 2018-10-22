// Copyright (c) 2018 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

package main

import (
	"fmt"

	"github.com/deanishe/go-safari/cloud"
)

func doListCloudTabs() error {

	c, err := cloud.New(cloud.DefaultTabsPath)
	if err != nil {
		return fmt.Errorf("couldn't open CloudTabs.db: %s", err)
	}

	tabs, err := c.Tabs()
	if err != nil {
		return fmt.Errorf("couldn't load cloud tabs: %s", err)
	}

	if outputJSON {
		return printJSON(tabs)
	}

	fStr := fmt.Sprintf("[%%%dd] %%s (%%s)\n", len(fmt.Sprintf("%d", len(tabs))))
	for i, tab := range tabs {
		fmt.Printf(fStr, i+1, tab.Title, tab.Device)
	}

	return nil
}
