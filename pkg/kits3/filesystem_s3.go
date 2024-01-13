package kits3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/h2non/filetype"
	"github.com/kitcat-framework/kitcat/kitstorage"
	"github.com/minio/minio-go/v7"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrAccessDenied      = errors.New("s3: access denied")
	ErrNoSuchBucket      = errors.New("s3: no such bucket")
	ErrInvalidBucketName = errors.New("s3: invalid bucket")
	ErrUnknownError      = errors.New("s3: unknown error")
)

type FileSystemS3 struct {
	client *minio.Client
	config Config
}

func NewFileStorageS3(client *minio.Client, config *Config) *FileSystemS3 {
	return &FileSystemS3{client: client, config: *config}
}

// Put uploads a file to the bucket
// The path must have at least a folder, it will be used as a bucket
func (f FileSystemS3) Put(ctx context.Context, filePath string, reader io.Reader, opts ...kitstorage.PutOptionFunc) error {
	o := kitstorage.NewPutOptions()

	for _, opt := range opts {
		opt(o)
	}

	bucket, file, err := f.getBucketName(filePath)
	if err != nil {
		return err
	}

	err = f.createBucketIfNotExists(bucket) // todo: add a local cache to avoid calling this method every time
	if err != nil {
		return fmt.Errorf("s3: unable to create bucket %s: %w", bucket, err)
	}

	teeReader := io.TeeReader(reader, new(bytes.Buffer))

	size, _ := io.Copy(io.Discard, teeReader)

	// Use the duplicated reader for further processing
	kind, _ := filetype.MatchReader(teeReader)

	fmt.Println("kind: ", kind.MIME.Value)
	fmt.Println("size: ", size)
	fmt.Println("file: ", file)
	fmt.Println("bucket: ", bucket)

	userMD := map[string]string{}

	if o.Public {
		userMD["x-amz-acl"] = "public-read"
	}

	// Reset the duplicated reader to the beginning
	if seeker, ok := teeReader.(io.ReadSeeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
	} else {
		return errors.New("Reader does not support seeking")
	}

	_, err = f.client.PutObject(ctx, bucket, file, reader, size, minio.PutObjectOptions{
		UserMetadata: userMD,
		ContentType:  kind.MIME.Value,
	})

	if err != nil {
		return fmt.Errorf("s3: unable to upload file %s: %w", filePath, err)
	}

	return nil
}

func (f FileSystemS3) Get(ctx context.Context, filePath string) (io.Reader, error) {
	bucket, file, err := f.getBucketName(filePath)
	if err != nil {
		return nil, err
	}

	obj, err := f.client.GetObject(ctx, bucket, file, minio.GetObjectOptions{})

	if err != nil {
		return nil, fmt.Errorf("s3: unable to get file %s: %w", filePath, err)
	}

	return obj, nil
}

func (f FileSystemS3) Exists(ctx context.Context, filePath string) (bool, error) {
	bucket, file, err := f.getBucketName(filePath)
	if err != nil {
		return false, err
	}

	_, err = f.client.StatObject(ctx, bucket, file, minio.StatObjectOptions{})

	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "AccessDenied" {
			return false, errors.Join(ErrAccessDenied, err)
		}
		if errResponse.Code == "NoSuchBucket" {
			return false, errors.Join(ErrNoSuchBucket, err)
		}
		if errResponse.Code == "InvalidBucketName" {
			return false, errors.Join(ErrInvalidBucketName, err)
		}
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, errors.Join(ErrUnknownError, err)
	}

	return true, nil
}

func (f FileSystemS3) Delete(ctx context.Context, filePath string) error {
	bucket, file, err := f.getBucketName(filePath)
	if err != nil {
		return err
	}

	err = f.client.RemoveObject(ctx, bucket, file, minio.RemoveObjectOptions{})

	if err != nil {
		return fmt.Errorf("s3: unable to delete file %s: %w", filePath, err)
	}

	return nil
}

func (f FileSystemS3) GetURL(ctx context.Context, filePath string, opts ...kitstorage.GetURLOptionFunc) (string, error) {
	o := kitstorage.NewGetURLOptions()

	for _, opt := range opts {
		opt(o)
	}

	bucket, file, err := f.getBucketName(filePath)
	if err != nil {
		return "", err
	}

	if o.PreSign {
		dur := time.Hour
		if o.Expiration != nil {
			dur = *o.Expiration
		}

		url, err := f.client.PresignedGetObject(ctx, bucket, file, dur, nil)

		if err != nil {
			return "", fmt.Errorf("s3: unable to get file url %s: %w", filePath, err)
		}

		return url.String(), nil
	}

	return fmt.Sprintf("https://%s", path.Join(f.config.Endpoint, bucket, file)), nil
}

func (f FileSystemS3) ListFiles(ctx context.Context, path string, recursive bool) ([]string, error) {
	bucket, file, err := f.getBucketName(path)
	if err != nil {
		return nil, err
	}

	var files []string

	if recursive {
		for obj := range f.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Prefix:    file,
			Recursive: true,
		}) {
			if obj.Err != nil {
				return nil, fmt.Errorf("s3: unable to list files %s: %w", path, err)
			}

			files = append(files, obj.Key)
		}
	} else {
		for obj := range f.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Prefix:    file,
			Recursive: false,
		}) {
			if obj.Err != nil {
				return nil, fmt.Errorf("s3: unable to list files %s: %w", path, err)
			}

			files = append(files, obj.Key)
		}
	}

	return files, nil
}

func (f FileSystemS3) Name() string {
	return "s3"
}

// findBucket returns the bucket and the file path
// dir1/dir2/file.txt -> dir1, dir2/file.txt
// dir1/file.txt -> dir1, file.txt
func (f FileSystemS3) findBucket(filePath string) (string, string) {
	dir, file := filepath.Split(filePath)
	dirs := filepath.SplitList(dir)

	if len(dirs) == 1 {
		return strings.TrimSuffix(dirs[0], "/"), file
	}

	return strings.TrimSuffix(dirs[0], "/"), filepath.Join(append(dirs[1:], file)...)
}

func (f FileSystemS3) createBucketIfNotExists(bucket string) error {
	exists, err := f.client.BucketExists(context.Background(), bucket)

	if err != nil {
		return fmt.Errorf("s3: unable to check if bucket %s exists: %w", bucket, err)
	}

	if !exists {
		err = f.client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{})

		if err != nil {
			return fmt.Errorf("s3: unable to create bucket %s: %w", bucket, err)
		}
	}

	return nil
}

func (f FileSystemS3) getBucketName(filePath string) (string, string, error) {
	dir := path.Dir(filePath)
	if dir == "" {
		return "", "", fmt.Errorf("s3: you must specify a folder to upload the file, it will be used as a bucket")
	}

	bucket, file := f.findBucket(filePath)
	return bucket, file, nil
}
