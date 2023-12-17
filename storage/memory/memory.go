package memory

import "net/url"

// tu or "tuple". A type to use in array backed storages (e.g. binary search, linear search)
type tu struct {
	from *url.URL
	to   *url.URL
}
