package s3

import (
	"errors"
	"fmt"
	"mime"
)

func (s *s3Service) validateUploadFile(data UploadFileData) error {
	if data.Filename == "" {
		return errors.New("filename is required")
	}

	if data.Base64Encoding == "" {
		return errors.New("base64Encoding is required")
	}

	if data.BucketName == "" {
		return errors.New("bucket name is required")
	}

	_, err := mime.ExtensionsByType(data.ContentType)
	if err != nil {
		return err
	}

	fileExist, err := s.isFileExist(data.BucketName, data.Filename)
	if err != nil {
		return err
	}

	if fileExist {
		return fmt.Errorf("file %s already exist on bucket %s", data.Filename, data.BucketName)
	}

	return nil
}

func (s *s3Service) validateDeleteFile(data DeleteFileData) error {
	if len(data.Filename) == 0 {
		return errors.New("filename is required")
	}

	if data.BucketName == "" {
		return errors.New("bucket name is required")
	}

	return nil
}
