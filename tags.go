package kilonova

import "log/slog"

type TagType string

const (
	TagTypeNone    TagType = ""
	TagTypeAuthor  TagType = "author"
	TagTypeContest TagType = "contest"
	TagTypeMethod  TagType = "method"
	TagTypeOther   TagType = "other"
)

type Tag struct {
	ID   int     `json:"id"`
	Name string  `json:"name"`
	Type TagType `json:"type"`
}

func (t *Tag) LogValue() slog.Value {
	if t == nil {
		return slog.Value{}
	}
	return slog.GroupValue(slog.Int("id", t.ID), slog.String("name", t.Name))
}

// Should be used for problem filtering
type TagGroup struct {
	// Negate instructs wether the filtered problem should have
	// or NOT have the corresponding tags in order for it to match
	Negate bool `json:"negate"`
	// TagIDs represents the set of tags which, when intersected with
	// the problem's tag set must be non-empty in order to get a match
	TagIDs []int `json:"tag_ids"`
}

func ValidTagType(t TagType) bool {
	return t == TagTypeAuthor || t == TagTypeContest ||
		t == TagTypeMethod || t == TagTypeOther
}
