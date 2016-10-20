package main

// This implements the necessary functions to satisfy the following interfaces needed for ftpserver:
// io.Writer, io.Reader, io.Closer, io.Seeker (stubbed out).  S3 manager requires only the io.Reader interface.

import (
	"bytes"
	"errors"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
)

type S3VirtualFile struct {
	s3session    *session.Session
	path         string       // the S3 key
	writeFile    bool         // only write to S3 if we've seen this flag
	writeBuffer  bytes.Buffer // a buffer that holds anything we want to write to a key on S3 (the value)
	s3FileIsRead bool
	readBuffer   *bytes.Buffer
	reader       *io.PipeReader
	writer       *io.PipeWriter
	s3FileOutput *s3.GetObjectOutput
	svc *s3.S3
}

func NewS3VirtualFile(path string) (*S3VirtualFile, error) {
	var err error
	f := &S3VirtualFile{path: path}

	f.s3session, err = session.NewSession()
	if err != nil {
		log.Println("error creating S3 session:", err)
		return nil, err
	}

	f.svc = s3.New(f.s3session)

	return f, nil
}

func (f *S3VirtualFile) Close() error {

	if f.writeFile {
		uploader := s3manager.NewUploader(f.s3session)
		upParams := &s3manager.UploadInput{
			Bucket: &S3_BUCKET_NAME,
			Key:    &f.path,
			// TODO would be nice to avoid buffering the file into memory
			Body: bytes.NewReader(f.writeBuffer.Bytes()),
		}

		_, err := uploader.Upload(upParams)

		f.writeFile = false
		return err
	}

	if f.s3FileIsRead {
		f.s3FileOutput.Body.Close()
	}

	return nil
}

func (f *S3VirtualFile) Read(buffer []byte) (int, error) {
	// Reading using s3.GetObject instead of s3manager.Downloader.  This won't work concurrently but it will let
	// us avoid reading the entire file in memory or on disk which is crucial.
	var err error
	if !f.s3FileIsRead {
		params := &s3.GetObjectInput{
			Bucket: &S3_BUCKET_NAME,
			Key:    &f.path,
		}

		if f.s3FileOutput, err = f.svc.GetObject(params); err != nil {
			return 0, err
		}

		f.s3FileIsRead = true
	}

	return f.s3FileOutput.Body.Read(buffer)
}

func (f *S3VirtualFile) Seek(n int64, w int) (int64, error) {
	// Can't seek to an S3 file.
	return 0, errors.New("Unable to seek to file in S3")
}

func (f *S3VirtualFile) Write(buffer []byte) (int, error) {
	f.writeFile = true
	return f.writeBuffer.Write(buffer)
}
