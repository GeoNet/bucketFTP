package main

// This implements the necessary functions to satisfy the following interfaces needed for ftpserver:
// io.Writer, io.Reader, io.Closer, io.Seeker (stubbed out, we won't use it).  S3 manager requires only the io.Reader interface.

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"os"
	"time"
)

type S3VirtualFile struct {
	flag         int
	s3Session    *session.Session
	s3Client     *s3.S3
	s3Path       string // the S3 key
	s3WriterOpen bool   // only write to S3 if we've seen this flag
	s3ReaderOpen bool
	s3FileOutput *s3.GetObjectOutput
	readPipe     *io.PipeReader
	writePipe    *io.PipeWriter
	uploadErr    chan error
}

func NewS3VirtualFile(path string, flag int, session *session.Session, client *s3.S3) (*S3VirtualFile, error) {
	f := &S3VirtualFile{
		flag:      flag,
		s3Path:    path,
		s3Session: session,
		s3Client:  client,
	}

	f.readPipe, f.writePipe = io.Pipe()
	f.uploadErr = make(chan error)

	// Set up the read/write objects at the start.  We get better ftp client errors if we can fail before reading or writing.
	var err error
	if flag == os.O_RDONLY {
		// read only doesn't need to modify the file
		params := &s3.GetObjectInput{
			Bucket: &S3_BUCKET_NAME,
			Key:    &f.s3Path,
		}

		if f.s3FileOutput, err = f.s3Client.GetObject(params); err != nil {
			return nil, stripNewlines(err)
		}

		f.s3ReaderOpen = true

	} else {

		// use PutObject to create an empty object.  This will report any errors before we write
		params := &s3.PutObjectInput{
			Bucket: &S3_BUCKET_NAME,
			Key:    &f.s3Path,
		}

		if _, err := f.s3Client.PutObject(params); err != nil {
			return nil, stripNewlines(err)
		}

		// using a go routine to avoid deadlock waiting on Write
		f.s3WriterOpen = true

		go func() {

			defer f.readPipe.Close()

			// Using s3manager because PutObject requires a ReadSeeker which we can't have with unbuffered input
			uploader := s3manager.NewUploader(f.s3Session)
			upParams := &s3manager.UploadInput{
				Bucket: &S3_BUCKET_NAME,
				Key:    &f.s3Path,
				Body:   f.readPipe,
			}

			_, err := uploader.Upload(upParams)
			f.uploadErr <- stripNewlines(err)

		}()
	}

	return f, nil
}

func (f *S3VirtualFile) Close() error {

	if f.s3ReaderOpen {
		f.s3FileOutput.Body.Close()
	}

	if f.s3WriterOpen {
		f.writePipe.Close()

		// waiting on this channel means we wait for the goroutine to finish uploading and check for error
		if err := <-f.uploadErr; err != nil {
			return err
		}
	}

	return nil
}

func (f *S3VirtualFile) Read(buffer []byte) (int, error) {
	if !f.s3ReaderOpen {
		return 0, errors.New("Unable to read from pipe")
	}

	return f.s3FileOutput.Body.Read(buffer)
}

func (f *S3VirtualFile) Seek(n int64, w int) (int64, error) {
	return 0, errors.New("Unable to seek in an S3 object")
}

func (f *S3VirtualFile) Write(buffer []byte) (int, error) {

	if !f.s3WriterOpen {
		return 0, errors.New("Unable to write to pipe")
	}

	return f.writePipe.Write(buffer)
}

type fakeInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	sys     interface{}
}

func (fi fakeInfo) Name() string {
	return fi.name
}

func (fi fakeInfo) Size() int64 {
	return fi.size
}

func (fi fakeInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi fakeInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi fakeInfo) IsDir() bool {
	return fi.isDir
}

func (fi fakeInfo) Sys() interface{} {
	return nil
}
