package server

import (
	"github.com/YannHulot/petstore/api/controllers"
	"github.com/YannHulot/petstore/api/repository"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// CreateRouter will create the routes and the router
func CreateRouter(db *gorm.DB) *gin.Engine {
	// force colors to show in terminal
	gin.ForceConsoleColor()

	// this router comes with a logger and a recovery middleware by default
	router := gin.Default()

	// version the api for future proofing and easy refactoring
	apiV1 := router.Group("/api/v1")

	// create a repository that gives access to the DB
	petRepository := repository.NewPetRepository(db)

	// create a controller that contains all the handlers that we need
	petController := controllers.NewPetController(petRepository)
	{
		apiV1.POST("/pet", petController.SavePet)
		apiV1.POST("/pet/:id", petController.UpdatePetWithFormData)
		apiV1.POST("/pet/:id/uploadImage", petController.UploadFile)
		apiV1.PUT("/pet", petController.UpdatePet)
		// WARNING: the route below handles multiple cases:
		// - pet/1
		// - pet/findByStatus?status=available
		// - pet/findByStatus?status=available&status=sold
		apiV1.GET("/pet/:id", petController.FindPetByIDOrStatus)
		apiV1.DELETE("/pet/:id", petController.DeletePet)
	}

	return router
}
