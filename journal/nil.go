package journal

type nilJournal struct{}

// nilj is a singleton nil journal.
var nilj Journal = &nilJournal{}

func NilJournal() Journal {
	return nilj
}

func (n *nilJournal) RegisterEntryType(_, _ string) EntryType { return EntryType{} }

func (n *nilJournal) RecordEntry(_ EntryType, _ interface{}) {}

func (n *nilJournal) Close() error { return nil }
