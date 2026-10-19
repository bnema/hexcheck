package domain

import "errors"

type ObjectID string

func (id ObjectID) String() string { return string(id) }
func (id ObjectID) IsZero() bool   { return id == "" }
func (id ObjectID) IsFull() bool   { return len(id) == 40 }

type Settings struct{ Enabled bool }

func (s Settings) IsZero() bool { return !s.Enabled }
func DefaultSettings() Settings { return Settings{Enabled: true} }

type Repo struct {
	Name     ObjectID
	Settings Settings
	Version  int
}
type Commit struct {
	ID      ObjectID
	Subject string
	Size    int
}

const MaxObjectBytes = 1024
const MaxTotalBytes = 4096

var ErrNotFound = errors.New("not found")
var ErrDisabled = errors.New("disabled")

func Allowed(id ObjectID) bool { return id.IsFull() }

type Limits struct {
	MaxLabel   string
	MaxEnabled bool
	Budget     int
}

var Budget = 4096
