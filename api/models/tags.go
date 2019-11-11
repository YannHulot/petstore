package models

// Tag is a tag for a pet
// ie: small, cute
type Tag struct {
	ID uint64 `gorm:"primary_key;not null;unique" json:"id"`
	// TagID uint64 `gorm:"primary_key" json:"id"`
	Name  string `json:"name"`
	PetID uint64 `json:"-"`
}
