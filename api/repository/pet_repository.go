package repository

import (
	"fmt"

	"github.com/YannHulot/petstore/api/models"
	"github.com/jinzhu/gorm"
)

// PetRepository provides access to the database
type PetRepository struct {
	datastore *gorm.DB
}

// NewPetRepository creates a new PetRepository
func NewPetRepository(db *gorm.DB) PetRepository {
	return PetRepository{
		datastore: db,
	}
}

// SavePet will save a pet in the database
func (p *PetRepository) SavePet(pet *models.Pet) (*models.Pet, error) {
	err := p.datastore.Debug().Model(&models.Pet{}).Create(&pet).Error
	if err != nil {
		return &models.Pet{}, err
	}

	return pet, nil
}

// FindPetByID will find a single Pet in the DB by its ID
func (p *PetRepository) FindPetByID(id string) (*models.Pet, error) {
	var pet models.Pet

	err := p.datastore.Debug().Preload("Tags").Preload("Category").First(&pet, id).Error
	if err != nil {
		return &pet, err
	}

	return &pet, nil
}

// UpdatePetAttributes will update a pet's name and status in the database
func (p *PetRepository) UpdatePetAttributes(id string, name string, status string) (*models.Pet, error) {
	err := p.datastore.Debug().Model(&models.Pet{}).Where("id = ?", id).Updates(
		map[string]interface{}{"name": name, "status": status}).Error
	if err != nil {
		return &models.Pet{}, err
	}

	pet, err := p.FindPetByID(id)
	if err != nil {
		return &models.Pet{}, err
	}

	return pet, nil
}

// UpdatePet will update a single pet in the database
func (p *PetRepository) UpdatePet(updatedPet *models.Pet) (*models.Pet, error) {
	if updatedPet.ID == 0 {
		return &models.Pet{}, fmt.Errorf("pet id is null, cannot update")
	}

	// update the main Pet record
	err := p.datastore.Debug().Model(&models.Pet{}).Updates(&updatedPet).Error
	if err != nil {
		return &models.Pet{}, err
	}

	// let's assume that the client sending the payload is the source of truth.
	// if we have a Pet record in a the DB with related Tag records, then we have to compare the Tags from the payload,
	// with the tags from the DB.

	// if the data from the DB Tag records is different, then update the Tag with the data from the payload,
	// if the TAG does not exists, then save the new Tag record in the DB

	// If there are no tags in the payload, then delete the Tag records  in the db.
	// if the number of tags in the payload is smaller than the number of tags in the DB then some tags need to be deleted.

	// Tt seems that in order to perform an update we need to do a lot of work to maintain data consistency
	// For the purpose of this example, we will simplify the process a bit.
	// We will delete all the related Tags and Category records and save whatever is in the payload.

	// WARNING: In a production environment, WE WOULD NOT DO THIS.

	// Delete all the related records
	err = p.datastore.Debug().Where("pet_id = ?", updatedPet.ID).Delete(&models.Tag{}).Error
	if err != nil {
		return &models.Pet{}, err
	}

	err = p.datastore.Debug().Where("pet_id = ?", updatedPet.ID).Delete(&models.Category{}).Error
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return &models.Pet{}, err
	}

	// create the new records
	category := updatedPet.Category
	tags := updatedPet.Tags

	for _, tag := range tags {
		// set up the association with teh pet record
		tag.PetID = updatedPet.ID
		err := p.datastore.Debug().Model(&models.Tag{}).Save(&tag).Error
		if err != nil {
			return &models.Pet{}, err
		}
	}

	// set up the association with the pet record
	category.PetID = updatedPet.ID
	err = p.datastore.Debug().Model(&models.Category{}).Save(&category).Error
	if err != nil {
		return &models.Pet{}, err
	}

	return updatedPet, nil
}

// FindPetByStatus will find pets by status
func (p *PetRepository) FindPetByStatus(status string) (*[]models.Pet, error) {
	var pets []models.Pet

	if len(status) == 0 {
		return &[]models.Pet{}, fmt.Errorf("status is empty. status is required to do the search")
	}

	err := p.datastore.Debug().
		Model(&models.Pet{}).
		Preload("Tags").
		Preload("Category").
		Where("status = ?", status).
		Limit(100).
		Find(&pets).Error
	if err != nil {
		return &[]models.Pet{}, err
	}

	return &pets, nil
}

// DeletePet will delete a pet in the database
func (p *PetRepository) DeletePet(id string) error {
	// cascading deletes
	// try to delete the related records first and then the main record

	err := p.datastore.Debug().Where("pet_id = ?", id).Delete(&models.Tag{}).Error
	if err != nil {
		return err
	}

	err = p.datastore.Debug().Where("pet_id = ?", id).Delete(&models.Category{}).Error
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return err
	}

	err = p.datastore.Debug().Where("id = ?", id).Delete(&models.Pet{}).Error
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return err
	}

	return nil
}
