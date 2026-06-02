package userconfig

// ApplyOptions controls how config sections are merged into the database.
type ApplyOptions struct {
	// MergeRadio inserts favourites and custom stations only when the UUID is
	// not already in the database. Existing rows are left unchanged.
	MergeRadio bool
}

// ReplaceOptions applies file values over matching database rows (upsert).
var ReplaceOptions = ApplyOptions{}

// MergeOptions adds radio entries from the file without updating existing UUIDs.
var MergeOptions = ApplyOptions{MergeRadio: true}
