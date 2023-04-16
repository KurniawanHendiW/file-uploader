package s3

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type S3Service interface {
	CreateBucket(bucketName string) error
	UploadFile(data UploadFileData) (string, error)
	DeleteFile(data DeleteFileData) error
}

type s3Service struct {
	region string
	s3Cli  *s3.Client
}

func NewS3Service(region string) S3Service {
	s3Svc := &s3Service{
		region: region,
	}

	if err := s3Svc.initSession(); err != nil {
		log.Fatalln(err)
	}

	return s3Svc
}

func (s *s3Service) initSession() error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	s.s3Cli = s3.NewFromConfig(cfg)

	return nil
}

func (s *s3Service) CreateBucket(bucketName string) error {
	if bucketName == "" {
		return errors.New("bucket name is required")
	}

	_, err := s.s3Cli.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.region),
		},
	})
	if err != nil {
		log.Printf("failed to create bucket %s: %v", bucketName, err)
		return err
	}

	return nil
}

func (s *s3Service) isExistBucket(bucketName string) (bool, error) {
	_, err := s.s3Cli.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				return false, nil
			default:
				log.Printf("don't have access to bucket %v or another error occurred: %v", bucketName, err)
				return false, err
			}
		}
	}

	return true, nil
}

func (s *s3Service) UploadFile(data UploadFileData) (string, error) {
	if err := s.validateUploadFile(data); err != nil {
		return "", err
	}

	pathFile := filepath.Base(fmt.Sprintf("%s/%s/%s", os.TempDir(), uuid.NewV4().String(), data.Filename))
	if err := createFile(data.Base64Encoding, pathFile); err != nil {
		return "", err
	}

	defer func() {
		if err := removeFile(pathFile); err != nil {
			log.Printf("failed to remove file %s: %v", pathFile, err)
		}
	}()

	file, err := os.Open(pathFile)
	if err != nil {
		return "", err
	}

	bucketExist, err := s.isExistBucket(data.BucketName)
	if err != nil {
		return "", err
	}

	if !bucketExist {
		if err = s.CreateBucket(data.BucketName); err != nil {
			return "", err
		}
	}

	var partMiBs int64 = 10
	uploader := manager.NewUploader(s.s3Cli, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
	})

	timeStartUpload := time.Now()
	output, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(data.BucketName),
		Key:         aws.String(data.Filename),
		ContentType: aws.String(data.ContentType),
		Body:        file,
	})
	log.Printf("upload file %s to bucket %s took %vs", data.Filename, data.BucketName, time.Since(timeStartUpload).Seconds())
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}

	return output.Location, nil
}

func (s *s3Service) isFileExist(bucketName, filename string) (bool, error) {
	_, err := s.s3Cli.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		var respErr *awsHttp.ResponseError
		if errors.As(err, &respErr) {
			if respErr.ResponseError.HTTPStatusCode() == http.StatusNotFound {
				return false, nil
			} else {
				log.Printf("get head object %s got error: %v", filename, respErr.Err.Error())
				return false, err
			}
		} else {
			log.Printf("don't have access to file %v or another error occurred: %v", filename, err)
			return false, err
		}
	}

	return true, nil
}

func createFile(fileBase64, pathFile string) error {
	dec, err := base64.StdEncoding.DecodeString(fileBase64)
	if err != nil {
		return err
	}

	file, err := os.Create(pathFile)
	if err != nil {
		return err
	}

	if _, err = file.Write(dec); err != nil {
		return err
	}

	return nil
}

func removeFile(pathFile string) error {
	if err := os.RemoveAll(pathFile); err != nil {
		return err
	}

	return nil
}

func (s *s3Service) DeleteFile(data DeleteFileData) error {
	if err := s.validateDeleteFile(data); err != nil {
		return err
	}

	fileExist := []string{}
	for _, filename := range data.Filename {
		isExist, err := s.isFileExist(data.BucketName, filename)
		if err != nil {
			return err
		}

		if isExist {
			fileExist = append(fileExist, filename)
		}
	}

	var objectIds []types.ObjectIdentifier
	for _, key := range fileExist {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(key)})
	}

	_, err := s.s3Cli.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(data.BucketName),
		Delete: &types.Delete{Objects: objectIds},
	})
	if err != nil {
		log.Printf("failed to delete files %v: %v", fileExist, err)
		return err
	}

	return nil
}
