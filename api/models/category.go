package models

// Category is a category of pet
type Category struct {
	ID         uint64 `gorm:"auto_increment" json:"-"`
	CategoryID uint64 `gorm:"primary_key" json:"id"`
	Name       string `json:"name"`
	PetID      uint64 `json:"-"`
}
