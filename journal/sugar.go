package journal

// MaybeRecordEvent is a convenience function that evaluates if the EntryType is
// enabled, and if so, it calls the supplier to create the event and
// subsequently journal.RecordEntry on the provided journal to record it.
//
// This is safe to call with a nil Journal, either because the value is nil,
// or because a journal obtained through NilJournal() is in use.
func MaybeRecordEvent(journal Journal, entryType EntryType, supplier func() interface{}) {
	if journal == nil || journal == nilj {
		return
	}
	if !entryType.Enabled() {
		return
	}
	journal.RecordEntry(entryType, supplier())
}
