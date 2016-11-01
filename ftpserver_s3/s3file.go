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
	s3Session    *session.Session
	s3Service    *s3.S3
	s3Path       string // the S3 key
	s3FileOpen   bool   // only write to S3 if we've seen this flag
	s3FileIsRead bool
	s3FileOutput *s3.GetObjectOutput
	readPipe     *io.PipeReader
	writePipe    *io.PipeWriter
	uploadErr    chan error
}

func NewS3VirtualFile(path string, session *session.Session, service *s3.S3) (*S3VirtualFile, error) {
	f := &S3VirtualFile{s3Path: path,
		s3Session: session,
		s3Service: service,
	}

	f.readPipe, f.writePipe = io.Pipe()

	f.uploadErr = make(chan error)

	return f, nil
}

func (f *S3VirtualFile) Close() error {

	if f.s3FileIsRead {
		f.s3FileOutput.Body.Close()
	}

	if f.s3FileOpen {
		f.writePipe.Close()

		// waiting on this channel means we wait for the goroutine to finish uploading and check for error
		if err := <-f.uploadErr; err != nil {
			return err
		}
	}

	return nil
}

func (f *S3VirtualFile) Read(buffer []byte) (int, error) {
	// Reading using s3.GetObject instead of s3manager.Downloader.  We could use WriteAtBuffer to buffer internally
	// but this approach avoids buffering of data on this ftp server.
	var err error
	if !f.s3FileIsRead {
		params := &s3.GetObjectInput{
			Bucket: &S3_BUCKET_NAME,
			Key:    &f.s3Path,
		}

		if f.s3FileOutput, err = f.s3Service.GetObject(params); err != nil {
			return 0, err
		}

		f.s3FileIsRead = true
	}

	return f.s3FileOutput.Body.Read(buffer)

}

func (f *S3VirtualFile) Seek(n int64, w int) (int64, error) {
	return 0, errors.New("Unable to seek in an S3 object")
}

func (f *S3VirtualFile) Write(buffer []byte) (int, error) {
	// Using reader and writer pipes to avoid buffering a file in memory.  Using a goroutine to do the S3 upload so
	// we can write to the pipe while the S3 call reads from it.
	if !f.s3FileOpen {
		f.s3FileOpen = true

		// using a go routine to avoid deadlock waiting on Write
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
			f.uploadErr <- err
		}()
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
