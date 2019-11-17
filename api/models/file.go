package models

import "mime/multipart"

// FileForm is the form that contains data about a file uploaded from a client
type FileForm struct {
	AdditionalMetadata string                `form:"additionalMetadata" binding:"required"`
	File               *multipart.FileHeader `form:"file" binding:"required"`
}
