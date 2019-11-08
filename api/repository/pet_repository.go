package repository

import (
	"fmt"

	"github.com/YannHulot/petstore/api/models"
	"github.com/jinzhu/gorm"
)

// PetRepository will give the repository access to the database
type PetRepository struct {
	datastore *gorm.DB
}

// NewPetRepository will generate a new repository
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

// FindPetByID will find a single Pet in the DB
func (p *PetRepository) FindPetByID(id string) (*models.Pet, error) {
	var pet models.Pet

	err := p.datastore.Debug().First(&pet, id).Related(&pet.Tags, "PetID").Related(&pet.Category, "PetID").Error
	if err != nil {
		return &pet, err
	}

	return &pet, nil
}

// UpdatePetAttributes will update a pet in the database
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

// UpdatePet will update a pet in the database
func (p *PetRepository) UpdatePet(updatedPet *models.Pet) (*models.Pet, error) {
	if updatedPet.ID == 0 {
		return &models.Pet{}, fmt.Errorf("pet id is null, cannot update")
	}

	// delete the old Tags records
	err := p.datastore.Debug().Where("pet_id = ?", updatedPet.ID).Delete(&models.Tag{}).Error
	if err != nil {
		return &models.Pet{}, err
	}

	// update the main Pet record
	err = p.datastore.Debug().Model(&models.Pet{}).Updates(&updatedPet).Error
	if err != nil {
		return &models.Pet{}, err
	}

	// update the related category record
	err = p.datastore.Debug().Model(&models.Category{}).Updates(&updatedPet.Category).Where("CategoryID", &updatedPet.Category.ID).Error
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

	err := p.datastore.Debug().Model(&models.Pet{}).Where("status = ?", status).Limit(100).Find(&pets).Error
	if err != nil {
		return &[]models.Pet{}, err
	}

	// I am aware that this is not the best way to do a find the related records but
	// I did not have the time to optimize every query
	if len(pets) > 0 {
		for i := range pets {
			err := p.datastore.Debug().Model(&models.Tag{}).Where("pet_id = ?", pets[i].ID).Take(&pets[i].Tags).Error
			if err != nil {
				if gorm.IsRecordNotFoundError(err) {
					// we don't want to return if an error occurs in case we can't find a related record,
					// we simply move on to the next record
					continue
				}
				return &[]models.Pet{}, err
			}

			err = p.datastore.Debug().Model(&models.Category{}).Where("pet_id = ?", pets[i].ID).Take(&pets[i].Category).Error
			if err != nil {
				if gorm.IsRecordNotFoundError(err) {
					// we don't want to return if an error occurs in case we can't find a related record,
					// we simply move on to the next record
					continue
				}
				return &[]models.Pet{}, err
			}
		}
	}

	return &pets, nil
}

// DeletePet will update a pet in the database
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
