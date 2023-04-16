package s3

import "errors"

var (
	ErrBucketNotFound = errors.New("bucket not found")
	ErrFileNotFound   = errors.New("file not found")
)

type (
	UploadFileRequest struct {
		BucketName     string
		ContentType    string
		Filename       string
		Base64Encoding string
	}

	DeleteFileRequest struct {
		BucketName string
		Filename   []string
	}

	DownloadFileRequest struct {
		BucketName string
		Filename   string
	}
)
