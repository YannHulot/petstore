package repository

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/YannHulot/petstore/api/models"
	"github.com/go-test/deep"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	DB   *gorm.DB
	mock sqlmock.Sqlmock

	repository PetRepository
}

func (s *Suite) SetupSuite() {
	var (
		db  *sql.DB
		err error
	)

	db, s.mock, err = sqlmock.New()
	require.NoError(s.T(), err)

	s.DB, err = gorm.Open("postgres", db)
	require.NoError(s.T(), err)

	s.DB.LogMode(true)

	s.repository = NewPetRepository(s.DB)
}

func (s *Suite) AfterTest(_, _ string) {
	require.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func TestInit(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) Test_repository_UpdatePetAttributes() {
	var (
		id     = "5"
		name   = "good-boy"
		status = "taken"
	)

	s.mock.ExpectBegin()

	s.mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "pets" SET "name" = $1, "status" = $2 WHERE (id = $3)`)).
		WithArgs(name, status, id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s.mock.ExpectCommit()

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE ("pets"."id" = $1) ORDER BY "pets"."id" ASC LIMIT 1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow(id, name, status))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "tags" WHERE ("pet_id" = $1) ORDER BY "tags"."tag_id" ASC`)).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(5, "mock-tag-name", 2))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories" WHERE ("pet_id" = $1) ORDER BY "categories"."category_id" ASC`)).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(5, "mock-category-name", 4))

	res, err := s.repository.UpdatePetAttributes(id, name, status)
	require.NoError(s.T(), err)
	require.Nil(s.T(), deep.Equal(&models.Pet{
		ID:     5,
		Name:   name,
		Status: status,
		Category: models.Category{
			ID:    4,
			Name:  "mock-category-name",
			PetID: 5,
		},
		Tags: []models.Tag{models.Tag{
			ID:    2,
			Name:  "mock-tag-name",
			PetID: 5,
		}},
	},
		res))
}

func (s *Suite) Test_repository_FindPetByID() {
	var (
		id   = "1"
		name = "doggy"
	)

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE ("pets"."id" = $1) ORDER BY "pets"."id" ASC LIMIT 1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(id, name))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "tags" WHERE ("pet_id" = $1) ORDER BY "tags"."tag_id" ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(1, "mock-tag-name", 2))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories" WHERE ("pet_id" = $1) ORDER BY "categories"."category_id" ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(1, "mock-category-name", 4))

	res, err := s.repository.FindPetByID(id)

	require.NoError(s.T(), err)

	expectedTag := models.Tag{
		Name:  "mock-tag-name",
		ID:    2,
		PetID: 1,
	}

	expectedCategory := models.Category{
		Name:  "mock-category-name",
		ID:    4,
		PetID: 1,
	}

	require.Nil(s.T(), deep.Equal(&models.Pet{
		ID:       1,
		Name:     name,
		Category: expectedCategory,
		Tags:     []models.Tag{expectedTag}},
		res))
}

func (s *Suite) Test_repository_FindPetByStatus() {
	var (
		id     = "1"
		name   = "doggy"
		status = "available"
	)

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE (status = $1) LIMIT 100`)).
		WithArgs(status).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow(id, name, status))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "tags"  WHERE (pet_id = $1) LIMIT 1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(1, "mock-tag-name", 2))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories"  WHERE (pet_id = $1) LIMIT 1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(1, "mock-category-name", 4))

	res, err := s.repository.FindPetByStatus(status)

	require.NoError(s.T(), err)

	expectedTag := models.Tag{
		Name:  "mock-tag-name",
		ID:    2,
		PetID: 1,
	}

	expectedCategory := models.Category{
		Name:  "mock-category-name",
		ID:    4,
		PetID: 1,
	}

	pet1 := models.Pet{
		ID:       1,
		Name:     name,
		Category: expectedCategory,
		Tags:     []models.Tag{expectedTag},
		Status:   status,
	}

	pets := []models.Pet{pet1}

	require.Nil(s.T(), deep.Equal(&pets, res))
}

func (s *Suite) Test_repository_SavePet() {
	var (
		name         = "doggy"
		status       = "available"
		categoryName = "mock-category-name"
		tagName      = "mock-tag-name"
	)

	expectedTag := models.Tag{
		Name:  tagName,
		PetID: 1,
	}

	expectedCategory := models.Category{
		Name:  categoryName,
		PetID: 1,
	}

	urls := pq.StringArray{"test"}

	pet1 := models.Pet{
		Name:       name,
		Category:   expectedCategory,
		Tags:       []models.Tag{expectedTag},
		PhotosURLs: urls,
		Status:     status,
	}

	s.mock.ExpectBegin()

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO "pets" ("name","photos_urls","status") VALUES ($1,$2,$3) RETURNING "pets"."id"`)).
		WithArgs(name, urls, status).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	s.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "categories" ("name","pet_id") VALUES ($1,$2) RETURNING "categories"."category_id"`)).
		WithArgs(categoryName, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

	s.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tags" ("name","pet_id") VALUES ($1,$2) RETURNING "tags"."tag_id"`)).
		WithArgs(tagName, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))

	s.mock.ExpectCommit()

	res, err := s.repository.SavePet(&pet1)

	require.NoError(s.T(), err)

	// value returned from the DB
	expectedCategory.ID = 0
	expectedCategory.CategoryID = 2
	expectedTag.ID = 0
	expectedTag.TagID = 3

	require.Nil(s.T(), deep.Equal(&models.Pet{
		ID:         1,
		Name:       name,
		Category:   expectedCategory,
		Tags:       []models.Tag{expectedTag},
		PhotosURLs: urls,
		Status:     status,
	},
		res))
}

func (s *Suite) Test_repository_UpdatePet() {
	var (
		name         = "doggy"
		status       = "available"
		categoryName = "mock-category-name"
		tagName      = "mock-tag-name"
	)

	expectedTag := models.Tag{
		Name:  tagName,
		ID:    2,
		PetID: 2,
	}

	expectedCategory := models.Category{
		Name:  categoryName,
		ID:    4,
		PetID: 2,
	}

	urls := pq.StringArray{"test"}

	pet1 := models.Pet{
		ID:         2,
		Name:       name,
		Category:   expectedCategory,
		Tags:       []models.Tag{expectedTag},
		PhotosURLs: urls,
		Status:     status,
	}

	s.mock.ExpectBegin()
	s.mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "tags" WHERE (pet_id = $1)`)).
		WithArgs(2).WillDelayFor(time.Second).
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectCommit()

	s.mock.ExpectBegin()
	s.mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "pets" SET "id" = $1, "name" = $2, "photos_urls" = $3, "status" = $4  WHERE "pets"."id" = $5`)).
		WithArgs(2, name, urls, status, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s.mock.ExpectQuery(regexp.QuoteMeta(`INSERT  INTO "categories" ("id","name","pet_id") VALUES ($1,$2,$3) RETURNING "categories"."category_id"`)).
		WithArgs(4, categoryName, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

	s.mock.ExpectQuery(regexp.QuoteMeta(`INSERT  INTO "tags" ("id","name","pet_id") VALUES ($1,$2,$3) RETURNING "tags"."tag_id"`)).
		WithArgs(2, tagName, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	s.mock.ExpectCommit()

	s.mock.ExpectBegin()
	s.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "categories" SET "id" = $1, "name" = $2, "pet_id" = $3  `)).
		WithArgs(4, categoryName, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectCommit()

	res, err := s.repository.UpdatePet(&pet1)

	require.NoError(s.T(), err)
	expectedTag.TagID = 2

	require.Nil(s.T(), deep.Equal(&models.Pet{
		ID:         2,
		Name:       name,
		Category:   expectedCategory,
		Tags:       []models.Tag{expectedTag},
		PhotosURLs: urls,
		Status:     status,
	},
		res))
}

func (s *Suite) Test_repository_DeletePet() {
	var (
		id = "2"
	)

	s.mock.ExpectBegin()
	s.mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "tags" WHERE (pet_id = $1)`)).
		WithArgs(id).WillDelayFor(time.Second).
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectCommit()

	s.mock.ExpectBegin()
	s.mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "categories" WHERE (pet_id = $1)`)).
		WithArgs(id).WillDelayFor(time.Second).
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectCommit()

	s.mock.ExpectBegin()
	s.mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "pets" WHERE (id = $1)`)).
		WithArgs(id).WillDelayFor(time.Second).
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectCommit()

	err := s.repository.DeletePet(id)
	require.NoError(s.T(), err)
}
