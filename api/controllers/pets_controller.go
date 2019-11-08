package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/YannHulot/petstore/api/models"
	"github.com/YannHulot/petstore/api/repository"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// PetController wraps all the o
type PetController struct {
	Repository repository.PetRepository
}

// NewPetController will create a new PetController
func NewPetController(repository repository.PetRepository) PetController {
	return PetController{
		Repository: repository,
	}
}

// UpdatePetWithFormData will update a pet in the database
func (p *PetController) UpdatePetWithFormData(c *gin.Context) {
	// get the param id(e.g 1) from the url
	id := c.Param("id")
	if id == "" {
		c.JSON(405, gin.H{"type": "error", "message": "Invalid input"})
		return
	}

	// get the form data from the request
	name := c.PostForm("name")
	status := c.PostForm("status")
	if len(name) == 0 && len(status) == 0 {
		c.JSON(405, gin.H{"type": "error", "message": "Invalid input"})
		return
	}

	updatedPet, err := p.Repository.UpdatePetAttributes(id, name, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "error", "message": "Invalid input"})
		return
	}

	c.JSON(http.StatusOK, updatedPet)
}

// UploadFile will save a file in storage
func (p *PetController) UploadFile(c *gin.Context) {
	id := c.Param("id")
	var form models.FileForm

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "error", "message": "Invalid Id param"})
		return
	}

	if err := c.ShouldBind(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "error", "message": "Invalid form value"})
		return
	}

	err := c.SaveUploadedFile(form.File, form.File.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "error", "message": err.Error()})
		return
	}

	message := fmt.Sprintf(
		"additionalMetadata: %s\nFile uploaded to ./%s, %v bytes",
		form.AdditionalMetadata, form.File.Filename, form.File.Size,
	)

	c.JSON(http.StatusOK, gin.H{"type": "unknown", "message": message})
}

// SavePet will save the pet in the database
func (p *PetController) SavePet(c *gin.Context) {
	var petToSave models.Pet

	err := c.ShouldBindJSON(&petToSave)
	if err != nil {
		log.Printf("failed parsing the body of the request: %v", err)
		c.JSON(405, gin.H{"type": "error", "message": "Invalid input"})
		return
	}

	// sanitise the data before saving
	petToSave.Sanitise()

	pet, err := p.Repository.SavePet(&petToSave)
	if err != nil {
		log.Printf("failed saving the pet in the db: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pet)
}

// FindPetByIDOrStatus will return a single Pet
func (p *PetController) FindPetByIDOrStatus(c *gin.Context) {
	// the id param is coming from the wildcard match in the router
	id := c.Param("id")
	isFindByStatus := strings.Contains(id, "findByStatus")

	// WARNING: this is a bit of a hack
	// gin router has some issues with wildcard in its pattern matching algorithm
	// so I had to be creative to follow the swagger template
	// see: https://github.com/gin-gonic/gin/issues/1301
	if isFindByStatus {
		p.findPetByStatus(c)
		return
	}

	p.findPetByID(c, id)
}

// findPetByID will find a pet based on its ID
func (p *PetController) findPetByID(c *gin.Context, id string) {
	pet, err := p.Repository.FindPetByID(id)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			log.Printf("failed to find the pet in the db: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"type": "error", "message": "invalid input"})
			return
		}
		log.Printf("failed to find the pet in the db: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"type": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pet)
}

// findPetByStatus will find a pet in the db based on its status
func (p *PetController) findPetByStatus(c *gin.Context) {
	var finalPets []models.Pet
	authorizedStatuses := map[string]bool{"sold": true, "available": true, "pending": true}

	statuses, valid := c.GetQueryArray("status")
	if !valid {
		log.Print("status is empty")
		c.JSON(400, gin.H{"type": "error", "message": "Invalid status value"})
		return
	}

	for _, status := range statuses {
		ok := authorizedStatuses[status]
		if !ok {
			log.Print("status is not authorized")
			c.JSON(400, gin.H{"type": "error", "message": "Invalid status value"})
			return
		}

		pets, err := p.Repository.FindPetByStatus(status)
		if err != nil {
			log.Printf("failed to find the pet in the db: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"type": "error", "message": err.Error()})
			return
		}

		finalPets = append(finalPets, *pets...)
	}

	c.JSON(http.StatusOK, finalPets)
}

// DeletePet will delete a single Pet
func (p *PetController) DeletePet(c *gin.Context) {
	apiKey := c.GetHeader("api_key")
	id := c.Param("id")

	if len(apiKey) == 0 {
		log.Print("API key not supplied")
		err := fmt.Errorf("Unauthorized - API key not supplied in Headers")
		c.JSON(http.StatusUnauthorized, gin.H{"type": "error", "message": err.Error()})
		return
	}

	err := p.Repository.DeletePet(id)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			log.Printf("failed to find the pet in the db: %v", err)
			c.JSON(404, gin.H{"type": "error", "message": "Pet not found"})
			return
		}
		log.Printf("failed to delete the pet or associated records in the db: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"type": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

// UpdatePet will update a single Pet
func (p *PetController) UpdatePet(c *gin.Context) {
	var petToSave models.Pet

	err := c.ShouldBindJSON(&petToSave)
	if err != nil {
		log.Printf("failed parsing the body of the request: %v", err)
		c.JSON(405, gin.H{"type": "Invalid input", "message": err.Error()})
		return
	}

	pet, err := p.Repository.UpdatePet(&petToSave)
	if err != nil {
		log.Printf("failed saving the pet in the db: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pet)
}
