package controllers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/YannHulot/petstore/api/models"
	"github.com/YannHulot/petstore/api/repository"
	"github.com/gin-gonic/gin"
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

	repository repository.PetRepository
	controller PetController
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

	petRepository := repository.NewPetRepository(s.DB)

	s.repository = petRepository
	s.controller = NewPetController(petRepository)
}

func (s *Suite) AfterTest(_, _ string) {
	require.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *Suite) assertJSON(actual []byte, data interface{}) {
	expected, err := json.Marshal(data)
	require.NoError(s.T(), err)

	if !bytes.Equal(expected, actual) {
		s.T().Errorf("the expected json: %s is different from actual %s", expected, actual)
	}
}

func TestInit(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) Test_controller_FindPetByIDOrStatus_findPetByID() {
	id := "1"
	name := "doggie"
	status := "available"

	r := gin.Default()
	r.GET("/api/v1/pet/:id", s.controller.FindPetByIDOrStatus)

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE ("pets"."id" = $1) ORDER BY "pets"."id" ASC LIMIT 1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow(id, name, status))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "tags" WHERE ("pet_id" IN ($1)) ORDER BY "tags"."id" ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(1, "mock-tag-name", 2))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories" WHERE ("pet_id" IN ($1)) ORDER BY "categories"."id" ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(1, "mock-category-name", 4))

	req, err := http.NewRequest("GET", "/api/v1/pet/1", nil)
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedPet := &models.Pet{
		ID:     1,
		Name:   name,
		Status: status,
		Category: models.Category{
			ID:    4,
			Name:  "mock-category-name",
			PetID: 8,
		},
		Tags: []models.Tag{models.Tag{
			ID:    2,
			Name:  "mock-tag-name",
			PetID: 8,
		}},
	}

	require.Nil(s.T(), deep.Equal(recorder.Code, 200))
	s.assertJSON(recorder.Body.Bytes(), expectedPet)

	if err := s.mock.ExpectationsWereMet(); err != nil {
		require.NoError(s.T(), err)
	}
}

func (s *Suite) Test_controller_FindPetByIDOrStatus_findPetByStatus_noStatus() {
	r := gin.Default()
	r.GET("/api/v1/pet/:id", s.controller.FindPetByIDOrStatus)

	req, err := http.NewRequest("GET", "/api/v1/pet/findByStatus?status=", nil)
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	errorResponse := `{"message":"Invalid status value","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Body.String(), errorResponse))
	require.Nil(s.T(), deep.Equal(recorder.Code, 400))
}

func (s *Suite) Test_controller_FindPetByIDOrStatus_findPetByStatus_invalidStatus() {
	r := gin.Default()
	r.GET("/api/v1/pet/:id", s.controller.FindPetByIDOrStatus)

	req, err := http.NewRequest("GET", "/api/v1/pet/findByStatus?status=test", nil)
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	errorResponse := `{"message":"Invalid status value","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Body.String(), errorResponse))
	require.Nil(s.T(), deep.Equal(recorder.Code, 400))
}

func (s *Suite) Test_controller_FindPetByIDOrStatus_findPetByStatus_errorInTransaction() {
	r := gin.Default()
	r.GET("/api/v1/pet/:id", s.controller.FindPetByIDOrStatus)
	status := "available"

	req, err := http.NewRequest("GET", "/api/v1/pet/findByStatus?status=available", nil)
	require.NoError(s.T(), err)

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE (status = $1) LIMIT 100`)).
		WithArgs(status).
		WillReturnError(fmt.Errorf("some error"))

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	errorResponse := `{"message":"some error","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Body.String(), errorResponse))
	require.Nil(s.T(), deep.Equal(recorder.Code, 500))

	req, err = http.NewRequest("GET", "/api/v1/pet/1", nil)
	require.NoError(s.T(), err)

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE ("pets"."id" = $1) ORDER BY "pets"."id" ASC LIMIT 1`)).
		WithArgs("1").
		WillReturnError(fmt.Errorf("another error"))

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	errorResponse = `{"message":"another error","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Body.String(), errorResponse))
	require.Nil(s.T(), deep.Equal(recorder.Code, 500))
}

func (s *Suite) Test_controller_FindPetByIDOrStatus_findPetByStatus_singleStatus() {
	id := "1"
	name := "doggie"
	status := "available"

	r := gin.Default()
	r.GET("/api/v1/pet/:id", s.controller.FindPetByIDOrStatus)

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE (status = $1) LIMIT 100`)).
		WithArgs(status).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow(id, name, status))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "tags" WHERE ("pet_id" IN ($1))`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(id, "mock-tag-name", 2))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories" WHERE ("pet_id" IN ($1))`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(id, "mock-category-name", 4))

	req, err := http.NewRequest("GET", "/api/v1/pet/findByStatus?status=available", nil)
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedPet := models.Pet{
		ID:     1,
		Name:   name,
		Status: status,
		Category: models.Category{
			ID:    4,
			Name:  "mock-category-name",
			PetID: 8,
		},
		Tags: []models.Tag{models.Tag{
			ID:    2,
			Name:  "mock-tag-name",
			PetID: 8,
		}},
	}
	expectedSlice := []models.Pet{expectedPet}

	s.assertJSON(recorder.Body.Bytes(), expectedSlice)
	require.Nil(s.T(), deep.Equal(recorder.Code, 200))

	if err := s.mock.ExpectationsWereMet(); err != nil {
		require.NoError(s.T(), err)
	}
}

func (s *Suite) Test_controller_FindPetByIDOrStatus_findPetByStatus_multipleStatuses() {
	id := "1"
	name := "doggie"
	status := "available"

	r := gin.Default()
	r.GET("/api/v1/pet/:id", s.controller.FindPetByIDOrStatus)

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE (status = $1) LIMIT 100`)).
		WithArgs(status).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow(id, name, status))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "tags" WHERE ("pet_id" IN ($1))`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(id, "mock-tag-name", 2))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories" WHERE ("pet_id" IN ($1))`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(id, "mock-category-name", 4))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "pets" WHERE (status = $1) LIMIT 100`)).
		WithArgs("sold").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow("7", "rover", "sold"))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "tags" WHERE ("pet_id" IN ($1))`)).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow("7", "mock-tag-name-2", 9))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories" WHERE ("pet_id" IN ($1))`)).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(7, "mock-category-name-2", 10))

	req, err := http.NewRequest("GET", "/api/v1/pet/findByStatus?status=available&status=sold", nil)
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedPet1 := models.Pet{
		ID:     1,
		Name:   name,
		Status: status,
		Category: models.Category{
			ID:    4,
			Name:  "mock-category-name",
			PetID: 8,
		},
		Tags: []models.Tag{models.Tag{
			ID:    2,
			Name:  "mock-tag-name",
			PetID: 8,
		}},
	}

	expectedPet2 := models.Pet{
		ID:     7,
		Name:   "rover",
		Status: "sold",
		Category: models.Category{
			ID:    10,
			Name:  "mock-category-name-2",
			PetID: 7,
		},
		Tags: []models.Tag{models.Tag{
			ID:    9,
			Name:  "mock-tag-name-2",
			PetID: 7,
		}},
	}

	expectedSlice := []models.Pet{expectedPet1, expectedPet2}
	s.assertJSON(recorder.Body.Bytes(), expectedSlice)
	require.Nil(s.T(), deep.Equal(recorder.Code, 200))

	if err := s.mock.ExpectationsWereMet(); err != nil {
		require.NoError(s.T(), err)
	}
}

func (s *Suite) createFileRequestBodyHelper(includeMetaData bool) (*bytes.Buffer, *multipart.Writer) {
	file, err := os.Open("../../filesForTest/test.png")
	require.NoError(s.T(), err)

	fileContents, err := ioutil.ReadAll(file)
	require.NoError(s.T(), err)

	fi, err := file.Stat()
	require.NoError(s.T(), err)
	file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fi.Name())
	require.NoError(s.T(), err)

	_, err = part.Write(fileContents)
	require.NoError(s.T(), err)

	if includeMetaData {
		err = writer.WriteField("additionalMetadata", "test")
		require.NoError(s.T(), err)
	}

	err = writer.Close()
	require.NoError(s.T(), err)

	return body, writer
}

func (s *Suite) Test_UploadFile_success() {
	body, writer := s.createFileRequestBodyHelper(true)

	r := gin.Default()
	r.POST("/api/v1/pet/:id/uploadImage", s.controller.UploadFile)

	req, err := http.NewRequest("POST", "/api/v1/pet/1/uploadImage", body)
	require.NoError(s.T(), err)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse := `{"message":"additionalMetadata: test\nFile uploaded to ./test.png, 419362 bytes","type":"unknown"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 200))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))
	if _, err := os.Stat("./test.png"); os.IsNotExist(err) {
		require.NoError(s.T(), err)
	}

	// clean up after test
	err = os.Remove("test.png")
	require.NoError(s.T(), err)
}

func (s *Suite) Test_UploadFile_error_no_metadata() {
	body, writer := s.createFileRequestBodyHelper(false)

	r := gin.Default()
	r.POST("/api/v1/pet/:id/uploadImage", s.controller.UploadFile)

	req, err := http.NewRequest("POST", "/api/v1/pet/1/uploadImage", body)
	require.NoError(s.T(), err)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse := `{"message":"Invalid form value","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 400))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))
	if _, err := os.Stat("./test.png"); os.IsExist(err) {
		require.NoError(s.T(), err)
	}
}

func (s *Suite) Test_UploadFile_error_no_id() {
	body, writer := s.createFileRequestBodyHelper(true)

	r := gin.Default()
	r.POST("/api/v1/pet/:id/uploadImage", s.controller.UploadFile)

	req, err := http.NewRequest("POST", "/api/v1/pet//uploadImage", body)
	require.NoError(s.T(), err)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	require.Nil(s.T(), deep.Equal(recorder.Code, 404))
}

func (s *Suite) Test_UpdatePetWithFormData_no_id() {
	r := gin.Default()
	r.POST("/api/v1/pet/:id", s.controller.UpdatePetWithFormData)

	req, err := http.NewRequest("POST", "/api/v1/pet/", nil)
	require.NoError(s.T(), err)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	require.Nil(s.T(), deep.Equal(recorder.Code, 404))
}

func (s *Suite) Test_UpdatePetWithFormData_invalid_form_params() {
	r := gin.Default()
	r.POST("/api/v1/pet/:id", s.controller.UpdatePetWithFormData)

	data := url.Values{}
	data.Set("name", "")
	data.Set("status", "")

	encodedData := strings.NewReader(data.Encode())

	req, err := http.NewRequest("POST", "/api/v1/pet/1", encodedData)
	require.NoError(s.T(), err)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse := `{"message":"Invalid input","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 405))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))
}

func (s *Suite) Test_UpdatePetWithFormData_success() {
	var (
		id     = "5"
		name   = "good-boy"
		status = "taken"
	)

	r := gin.Default()
	r.POST("/api/v1/pet/:id", s.controller.UpdatePetWithFormData)

	data := url.Values{}
	data.Set("name", name)
	data.Set("status", status)

	encodedData := strings.NewReader(data.Encode())

	req, err := http.NewRequest("POST", "/api/v1/pet/5", encodedData)
	require.NoError(s.T(), err)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

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
		`SELECT * FROM "tags" WHERE ("pet_id" IN ($1)) ORDER BY "tags"."id`)).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(5, "mock-tag-name", 2))

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "categories" WHERE ("pet_id" IN ($1)) ORDER BY "categories"."id`)).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "id"}).
			AddRow(5, "mock-category-name", 4))

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse := `{"id":5,"category":{"id":4,"name":"mock-category-name"},"name":"good-boy","photoUrls":null,"tags":[{"id":2,"name":"mock-tag-name"}],"status":"taken"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 200))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))
}

func (s *Suite) Test_DeletePet_error_no_id() {
	r := gin.Default()
	r.DELETE("/api/v1/pet/:id", s.controller.DeletePet)

	req, err := http.NewRequest("DELETE", "/api/v1/pet/", nil)
	require.NoError(s.T(), err)
	req.Header.Add("api_key", "test-key")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	require.Nil(s.T(), deep.Equal(recorder.Code, 404))
}

func (s *Suite) Test_DeletePet_error_no_api_key() {
	r := gin.Default()
	r.DELETE("/api/v1/pet/:id", s.controller.DeletePet)

	req, err := http.NewRequest("DELETE", "/api/v1/pet/1", nil)
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse := `{"message":"unauthorized - API key not supplied in Headers","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 401))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))
}

func (s *Suite) Test_DeletePet_error_pet_not_found() {
	var (
		id = "2"
	)

	r := gin.Default()
	r.DELETE("/api/v1/pet/:id", s.controller.DeletePet)

	req, err := http.NewRequest("DELETE", "/api/v1/pet/2", nil)
	require.NoError(s.T(), err)
	req.Header.Add("api_key", "test-key")

	s.mock.ExpectBegin()
	s.mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "tags" WHERE (pet_id = $1)`)).
		WithArgs(id).WillDelayFor(time.Second).
		WillReturnError(gorm.ErrRecordNotFound)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse := `{"message":"Pet not found","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 404))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))
}

func (s *Suite) Test_DeletePet_success() {
	var (
		id = "2"
	)

	r := gin.Default()
	r.DELETE("/api/v1/pet/:id", s.controller.DeletePet)

	req, err := http.NewRequest("DELETE", "/api/v1/pet/2", nil)
	require.NoError(s.T(), err)
	req.Header.Add("api_key", "test-key")

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

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	require.Nil(s.T(), deep.Equal(recorder.Code, 200))
}

func (s *Suite) Test_SavePet_error_invalid_payload() {
	r := gin.Default()
	r.POST("/api/v1/pet", s.controller.SavePet)

	badPet := models.Pet{
		Status: "test",
	}

	payload, err := json.Marshal(badPet)
	require.NoError(s.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/pet", strings.NewReader(string(payload)))
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse := `{"message":"Invalid input","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 405))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))

	badPet.Name = "Rover"
	badPet.Status = ""

	req, err = http.NewRequest("POST", "/api/v1/pet", strings.NewReader(string(payload)))
	require.NoError(s.T(), err)

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	expectedResponse = `{"message":"Invalid input","type":"error"}`

	require.Nil(s.T(), deep.Equal(recorder.Code, 405))
	require.Nil(s.T(), deep.Equal(recorder.Body.String(), expectedResponse))
}

func (s *Suite) Test_SavePet_success() {
	var (
		name         = "doggy"
		status       = "available"
		categoryName = "mock-category-name"
		tagName      = "mock-tag-name"
	)

	r := gin.Default()
	r.POST("/api/v1/pet", s.controller.SavePet)

	expectedTag := models.Tag{
		ID:    6,
		Name:  tagName,
		PetID: 1,
	}

	expectedCategory := models.Category{
		ID:    12,
		Name:  categoryName,
		PetID: 1,
	}

	urls := pq.StringArray{"test"}

	goodPet := models.Pet{
		Name:       name,
		Category:   expectedCategory,
		Tags:       []models.Tag{expectedTag},
		PhotosURLs: urls,
		Status:     status,
	}

	payload, err := json.Marshal(goodPet)
	require.NoError(s.T(), err)

	s.mock.ExpectBegin()

	s.mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO "pets" ("name","photos_urls","status") VALUES ($1,$2,$3) RETURNING "pets"."id"`)).
		WithArgs(name, urls, status).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	s.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "categories" SET "name" = $1, "pet_id" = $2  WHERE "categories"."id" = $3`)).
		WithArgs(categoryName, 1, 12).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tags" SET "name" = $1, "pet_id" = $2  WHERE "tags"."id" = $3`)).
		WithArgs(tagName, 1, 6).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s.mock.ExpectCommit()

	req, err := http.NewRequest("POST", "/api/v1/pet", strings.NewReader(string(payload)))
	require.NoError(s.T(), err)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	savedPet := &models.Pet{}
	err = json.Unmarshal(recorder.Body.Bytes(), savedPet)
	require.NoError(s.T(), err)

	// expected values
	goodPet.ID = 1
	goodPet.Category.PetID = 0
	goodPet.Tags[0].PetID = 0

	require.Nil(s.T(), deep.Equal(recorder.Code, 200))
	require.Nil(s.T(), deep.Equal(*savedPet, goodPet))
}
