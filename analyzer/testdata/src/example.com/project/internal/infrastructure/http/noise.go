package http

import (
	"errors"
	"example.com/project/internal/domain"
	"strings"
)

type storedRepo struct {
	Name    string
	Version int
	Enabled bool
}

func decodeRepo(raw storedRepo) domain.Repo {
	var repo domain.Repo
	repo.Name = domain.ObjectID(raw.Name)
	repo.Version = raw.Version
	repo.Settings.Enabled = raw.Enabled
	if raw.Name == "" {
		repo.Name = "unknown"
	}
	return repo
}
func parseCommit(output string, id domain.ObjectID) domain.Commit {
	var c domain.Commit
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "subject ") {
			c.Subject = strings.TrimPrefix(line, "subject ")
		}
		if line == id.String() {
			c.ID = id
		}
	}
	return c
}

var libraryNotFound = errors.New("missing")

func translateError(err error) error {
	if errors.Is(err, libraryNotFound) {
		return domain.ErrNotFound
	}
	return err
}
func Update(repo domain.Repo, version int, put func(string, domain.Repo) error) error {
	if repo.Version != version {
		return domain.ErrNotFound
	}
	if err := put(repo.Name.String(), repo); err != nil {
		return translateError(err)
	}
	return nil
}

func resolveRevision(rev string, id domain.ObjectID, lookup func(string) bool) string { // want "suspicious-business-logic-in-adapter"
	if rev == "HEAD" {
		return "head"
	}
	if id.IsFull() {
		if lookup(id.String()) {
			return "object"
		}
		if lookup("heads/" + rev) {
			return "branch"
		}
		if lookup("tags/" + rev) {
			return "tag"
		}
	}
	if strings.HasPrefix(rev, "refs/") {
		return rev
	}
	if !id.IsZero() && lookup(rev) {
		return "abbrev"
	}
	return ""
}
func ReadCommitObjects(items []domain.Commit) []domain.Commit { // want "suspicious-business-logic-in-adapter"
	var out []domain.Commit
	total := 0
	for _, c := range items {
		if c.Size > domain.MaxObjectBytes {
			continue
		}
		if total+c.Size > domain.MaxTotalBytes {
			break
		}
		total += c.Size
		out = append(out, c)
	}
	return out
}
func commitsAhead(source, target []domain.Commit, tip string) ([]domain.Commit, bool) { // want "suspicious-business-logic-in-adapter"
	seen := map[string]bool{}
	for _, c := range target {
		seen[c.ID.String()] = true
	}
	var ahead []domain.Commit
	for _, c := range source {
		if seen[c.ID.String()] {
			return ahead, c.ID.String() != tip
		}
		if len(ahead) > 100 {
			break
		}
		ahead = append(ahead, c)
	}
	return ahead, true
}
func repoRecord(get func() (domain.Repo, error)) (domain.Repo, error) { // want "suspicious-business-logic-in-adapter"
	repo, err := get()
	if err != nil {
		return domain.Repo{}, err
	}
	if repo.Settings.IsZero() {
		repo.Settings = domain.DefaultSettings()
	}
	return repo, nil
}
func CreateRun(disabled bool, save func() error) error { // want "suspicious-business-logic-in-adapter"
	if disabled {
		return domain.ErrDisabled
	}
	if err := save(); err != nil {
		return err
	}
	return nil
}

//hexcheck:ignore suspicious-business-logic-in-adapter decoding has no policy // want "unused-ignore-directive"
func cleanDecode(raw storedRepo) domain.Repo { return decodeRepo(raw) }
