package journal

import (
	"sync"
	"time"
)

// DisabledEntries is the set of event types whose journaling is suppressed.
type DisabledEntries []EntryType

// EntryType represents the signature of an event.
type EntryType struct {
	System string
	Event  string

	// enabled stores whether this event type is enabled.
	enabled bool

	// safe is a sentinel marker that's set to true if this EntryType was
	// constructed correctly (via Journal#RegisterEntryType).
	safe bool
}

// Enabled returns whether this event type is enabled in the journaling
// subsystem. Users are advised to check this before actually attempting to
// add a journal entry, as it helps bypass object construction for events that
// would be discarded anyway.
//
// All event types are enabled by default, and specific event types can only
// be disabled at Journal construction time.
func (et EntryType) Enabled() bool {
	return et.enabled
}

// Journal represents an audit trail of system actions.
//
// Every entry is tagged with a timestamp, a system name, and an event name.
// The supplied data can be any type, as long as it is JSON serializable,
// including structs, map[string]interface{}, or primitive types.
//
// For cleanliness and type safety, we recommend to use typed events. See the
// *Evt struct types in this package for more info.
type Journal interface {
	// RegisterEntryType introduces a new entry type to this journal, and
	// returns an EntryType token that components can later use to check whether
	// journalling for that type is enabled/suppressed, and to tag journal
	// entries appropriately.
	RegisterEntryType(system, event string) EntryType

	// RecordEntry records this entry to the journal.
	//
	// See godocs on the Journal type for more info.
	RecordEntry(entryType EntryType, data interface{})

	// Close closes this journal for further writing.
	Close() error
}

// Event represents a journal entry.
//
// See godocs on Journal for more information.
type Event struct {
	EntryType

	Timestamp time.Time
	Data      interface{}
}

// entryTypeFactory is an embeddable mixin that takes care of tracking disabled
// event types, and returning initialized/safe EventTypes when requested.
type entryTypeFactory struct {
	sync.Mutex

	m map[string]EntryType
}

func newEntryTypeFactory(disabled DisabledEntries) *entryTypeFactory {
	ret := &entryTypeFactory{
		m: make(map[string]EntryType, len(disabled)+32), // + extra capacity.
	}

	for _, et := range disabled {
		et.enabled, et.safe = false, true
		ret.m[et.System+":"+et.Event] = et
	}

	return ret
}

func (d *entryTypeFactory) RegisterEntryType(system, event string) EntryType {
	d.Lock()
	defer d.Unlock()

	key := system + ":" + event
	if et, ok := d.m[key]; ok {
		return et
	}

	et := EntryType{
		System:  system,
		Event:   event,
		enabled: true,
		safe:    true,
	}

	d.m[key] = et
	return et
}
