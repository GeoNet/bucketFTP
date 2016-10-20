package main

// This implements the necessary functions to satisfy the following interfaces needed for ftpserver:
// io.Writer, io.Reader, io.Closer, io.Seeker (stubbed out).  S3 manager requires only the io.Reader interface.

import (
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
	s3FileOpen   bool         // only write to S3 if we've seen this flag
	s3FileIsRead bool
	reader       *io.PipeReader
	writer       *io.PipeWriter
	s3FileOutput *s3.GetObjectOutput
	s3Service    *s3.S3
}

func NewS3VirtualFile(path string) (*S3VirtualFile, error) {
	var err error
	f := &S3VirtualFile{path: path}

	f.s3session, err = session.NewSession()
	if err != nil {
		log.Println("error creating S3 session:", err)
		return nil, err
	}

	f.s3Service = s3.New(f.s3session)

	f.reader, f.writer = io.Pipe()

	return f, nil
}

func (f *S3VirtualFile) Close() error {

	if f.s3FileIsRead {
		f.s3FileOutput.Body.Close()
	}

	f.writer.Close()

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

		if f.s3FileOutput, err = f.s3Service.GetObject(params); err != nil {
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
	// Using reader and writer pipes to avoid buffering a file in memory
	if !f.s3FileOpen {
		f.s3FileOpen = true

		// using a go routine to avoid deadlock waiting on Write
		go func() {

			defer f.reader.Close()

			uploader := s3manager.NewUploader(f.s3session)
			upParams := &s3manager.UploadInput{
				Bucket: &S3_BUCKET_NAME,
				Key:    &f.path,
				Body:   f.reader,
			}

			if _, err := uploader.Upload(upParams); err != nil {
				log.Println("error uploading file", err)
				//return
				// TODO: find a way to abort if the upload fails (hard since we're in a goroutine).  Fatal??
			}
		}()
	}

	return f.writer.Write(buffer)
}
