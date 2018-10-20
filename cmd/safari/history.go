//
// Copyright (c) 2018 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2018-08-24
//

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/deanishe/go-safari/history"
)

func doSearchHistory() error {

	history.MaxSearchResults = 20

	h, err := history.New(history.DefaultHistoryPath)
	if err != nil {
		return err
	}

	if searchQuery == "" {
		fmt.Fprintln(os.Stderr, "search query is empty")
		return nil
	}

	fmt.Fprintf(os.Stderr, "searching for %q ...\n", searchQuery)

	entries, err := h.Search(searchQuery)
	if err != nil {
		return err
	}

	if outputJSON {
		data, err := json.Marshal(entries)
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, string(data))
		return nil
	}

	for i, e := range entries {
		fmt.Printf("[%d/%d] %q (%s)\n", i+1, len(entries), e.Title, e.URL)
	}

	return nil
}
