package store

import (
	"time"
)

type UnfinishedEntry struct {
	Title string
	Start time.Time
	Notes string
}

type Store interface {
	Start(*UnfinishedEntry) error
	StartTitle(string, string) error
	Finish(string) error
}

func NewSqliteStore(db string) (Store, error) {
	s, err := newSqlite(db)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
