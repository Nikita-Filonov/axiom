package testflags

import (
	"flag"
	"slices"
)

// Flag describes one registered flag at snapshot time. Default is the original
// textual default; Value is the parsed value. Set reports whether the flag was
// explicitly set, including through flag.Set, even when Value equals Default.
type Flag struct {
	Set     bool
	Name    string
	Value   any
	Usage   string
	Default string
}

// Flags is a read-only snapshot. Its zero value is empty. All source flags are
// included, including Go's testing flags when using flag.CommandLine. Values
// come from flag.Getter when available, otherwise from flag.Value.String.
// Custom Getter values are shallow copies; their owners must not mutate shared
// values after capture. Built-in flag values need no additional synchronization.
type Flags struct {
	entries map[string]Flag
}

// Get returns the untyped value and whether name exists.
func (f *Flags) Get(name string) (any, bool) {
	entry, ok := f.entries[name]
	return entry.Value, ok
}

// Lookup returns a copy of the flag's value and metadata, or false if absent.
func (f *Flags) Lookup(name string) (Flag, bool) {
	entry, ok := f.entries[name]
	return entry, ok
}

// All returns metadata sorted by name in a new slice. Values follow the shallow
// copy rule of Flags; changing the slice or its entry fields does not change f.
func (f *Flags) All() []Flag {
	names := make([]string, 0, len(f.entries))
	for name := range f.entries {
		names = append(names, name)
	}
	slices.Sort(names)
	entries := make([]Flag, 0, len(names))
	for _, name := range names {
		entries = append(entries, f.entries[name])
	}
	return entries
}

func snapshot(fs *flag.FlagSet) *Flags {
	result := &Flags{entries: make(map[string]Flag)}
	fs.VisitAll(func(f *flag.Flag) {
		var value any
		if getter, ok := f.Value.(flag.Getter); ok {
			value = getter.Get()
		} else {
			value = f.Value.String()
		}
		result.entries[f.Name] = Flag{Name: f.Name, Value: value, Usage: f.Usage, Default: f.DefValue}
	})
	fs.Visit(func(f *flag.Flag) {
		entry := result.entries[f.Name]
		entry.Set = true
		result.entries[f.Name] = entry
	})
	return result
}
