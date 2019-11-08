package models

// Tag is a tag for a pet
// ie: small, cute
type Tag struct {
	ID    uint64 `gorm:"auto_increment" json:"-"`
	TagID uint64 `gorm:"primary_key" json:"id"`
	Name  string `json:"name"`
	PetID uint64 `json:"-"`
}
