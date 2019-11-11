package models

// Category is a category of pet
type Category struct {
	ID    uint64 `gorm:"primary_key;not null;unique" json:"id"`
	Name  string `json:"name"`
	PetID uint64 `json:"-"`
}
