package models

import (
	"html"
	"strings"

	"github.com/lib/pq"
)

// Pet represent a pet saved in our store
type Pet struct {
	ID         uint64         `gorm:"primary_key;auto_increment" json:"id"`
	Category   Category       `gorm:"foreignkey:PetID" json:"category"`
	Name       string         `gorm:"size:255;not null;" json:"name" binding:"required"`
	PhotosURLs pq.StringArray `gorm:"type:varchar(100)[]" json:"photoUrls"`
	Tags       []Tag          `gorm:"foreignkey:PetID" json:"tags"`
	Status     string         `json:"status"`
}

// Sanitise will sanitise the values that will be saved in the database
func (p *Pet) Sanitise() {
	p.Name = html.EscapeString(strings.TrimSpace(p.Name))
	p.Status = html.EscapeString(strings.TrimSpace(p.Status))

	p.Category.Name = html.EscapeString(strings.TrimSpace(p.Category.Name))

	if len(p.Tags) > 0 {
		for i := range p.Tags {
			p.Tags[i].Name = html.EscapeString(strings.TrimSpace(p.Tags[i].Name))
			p.Tags[i].PetID = 0
		}
	}

	if len(p.PhotosURLs) > 0 {
		for i := range p.PhotosURLs {
			p.PhotosURLs[i] = html.EscapeString(strings.TrimSpace(p.PhotosURLs[i]))
		}
	}
}
