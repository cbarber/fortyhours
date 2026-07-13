package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
)

// printTable writes rows as a tab-aligned table with the given header.
func printTable(w io.Writer, header []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, joinTabs(header))
	for _, row := range rows {
		fmt.Fprintln(tw, joinTabs(row))
	}
	return tw.Flush()
}

func joinTabs(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += "\t"
		}
		out += c
	}
	return out
}

// printJSON writes v as indented JSON.
func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// str dereferences a *string, returning "" for nil.
func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// intStr renders a *int as a string, or "" for nil.
func intStr(p *int) string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(*p)
}

// boolStr renders a *bool as a string, or "" for nil.
func boolStr(p *bool) string {
	if p == nil {
		return ""
	}
	return strconv.FormatBool(*p)
}
