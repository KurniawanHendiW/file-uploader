package s3

type (
	UploadFileData struct {
		BucketName     string
		ContentType    string
		Filename       string
		Base64Encoding string
	}

	DeleteFileData struct {
		BucketName string
		Filename   []string
	}
)
