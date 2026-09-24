package storage

import "sort"

// SortLinks gives list results a stable order across storage backends.
func SortLinks(links []Link) {
	sort.Slice(links, func(i, j int) bool {
		return links[i].From.String() < links[j].From.String()
	})
}
