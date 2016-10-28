package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fclairamb/ftpserver/server"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SampleDriver defines a very basic serverftp driver
type S3Driver struct {
	baseDir   string
	s3Session *session.Session
	s3Service *s3.S3
}

func (d *S3Driver) WelcomeUser(cc server.ClientContext) (string, error) {
	cc.SetDebug(true)
	return "Welcome to the FTP server for S3", nil
}

func (d *S3Driver) AuthUser(cc server.ClientContext, user, pass string) (server.ClientHandlingDriver, error) {

	if user != FTP_USER {
		log.Println("username does not match expected user", user)
		return nil, fmt.Errorf("incorrect username: %s", user)
	}

	if pass != FTP_PASSWD {
		log.Println("incorrect password")
		return nil, errors.New("incorrect password")
	}

	_, err := session.NewSession()
	if err != nil {
		log.Println("error creating S3 session (check AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY env vars on server):", err)
		return nil, err
	}

	return d, nil
}

func (d *S3Driver) GetTLSConfig() (*tls.Config, error) {
	return nil, errors.New("TLS not implemented")
}

func (d *S3Driver) ChangeDirectory(cc server.ClientContext, directory string) error {
	// Query S3 to see if the directory exists.  If it does update the baseDir variable to emulate cd-ing to that
	// directory

	var err error
	var dirname string

	// join paths and keep relative to root dir.
	if dirname, err = filepath.Rel("/", directory); err != nil {
		return err
	}

	if dirname == "." {
		dirname = ""
	}

	// directories in S3 must always have a trailing slash (filepath.Rel removes this)
	if len(dirname) > 0 && !strings.HasSuffix(dirname, "/") {
		dirname += "/"
	}

	delimiter := "/"
	params := &s3.ListObjectsV2Input{
		Bucket:    &S3_BUCKET_NAME,
		Prefix:    &dirname,
		Delimiter: &delimiter, // limits the search to directories (they have a trailing slash)
	}

	var resp *s3.ListObjectsV2Output
	if resp, err = d.s3Service.ListObjectsV2(params); err != nil {
		return err
	}

	if *resp.KeyCount == 0 {
		return errors.New("No such directory: " + directory)
	}

	d.baseDir = dirname

	return nil
}

func (d *S3Driver) MakeDirectory(cc server.ClientContext, directory string) error {

	dirname := directory
	if !strings.HasSuffix(dirname, "/") {
		dirname += "/"
	}

	params := &s3.PutObjectInput{
		Bucket: &S3_BUCKET_NAME,
		Key:    &dirname,
		Body:   bytes.NewReader([]byte("")),
	}

	var err error
	if _, err = d.s3Service.PutObject(params); err != nil {
		return err
	}

	return nil
}

func (d *S3Driver) ListFiles(cc server.ClientContext) ([]os.FileInfo, error) {

	delimiter := "/" // delimiter keeps the listing from being recursive
	params := &s3.ListObjectsV2Input{
		Bucket:    &S3_BUCKET_NAME,
		Prefix:    &d.baseDir,
		Delimiter: &delimiter,
	}

	var err error
	var resp *s3.ListObjectsV2Output
	if resp, err = d.s3Service.ListObjectsV2(params); err != nil {
		return nil, err
	}

	files := []os.FileInfo{}

	// directories other than CWD
	for _, dir := range resp.CommonPrefixes {
		var dirInfo os.FileInfo

		if dirInfo, err = d.GetFileInfo(cc, filepath.Join("/", *dir.Prefix)+"/"); err != nil {
			return nil, err
		}

		files = append(files, dirInfo)
	}

	// files and CWD
	for _, f := range resp.Contents {

		// don't list CWD in the list of files
		if *f.Key == d.baseDir {
			continue
		}

		var fi os.FileInfo
		if fi, err = d.fakeFileInfo(*f.Key, *f.Size, *f.LastModified); err != nil {
			return nil, err
		}
		files = append(files, fi)
	}

	return files, nil
}

func (d *S3Driver) UserLeft(cc server.ClientContext) {

}

func (d *S3Driver) OpenFile(cc server.ClientContext, path string, flag int) (server.FileStream, error) {
	// our implementation uses an interface that mimics a file but uploads/downloads to S3
	var err error
	var s3file *S3VirtualFile

	if s3file, err = NewS3VirtualFile(path, d.s3Session, d.s3Service); err != nil {
		return nil, err
	}

	return s3file, nil
}

func (d *S3Driver) GetFileInfo(cc server.ClientContext, path string) (os.FileInfo, error) {

	var err error
	var relPath string
	if relPath, err = filepath.Rel("/", path); err != nil {
		return nil, err
	}

	// dirs need to have a trailing slash
	if strings.HasSuffix(path, "/") {
		relPath += "/"
	}

	params := &s3.GetObjectInput{
		Bucket: &S3_BUCKET_NAME,
		Key:    &relPath,
	}

	req, resp := d.s3Service.GetObjectRequest(params)

	if err := req.Send(); err != nil {
		return nil, err
	}

	// resp.ContentLength and LastModified are sometimes nil (!) so check for this state
	var objectSize int64
	if resp.ContentLength != nil {
		objectSize = *resp.ContentLength
	}

	var modTime time.Time
	if resp.LastModified != nil {
		modTime = *resp.LastModified
	}

	var f os.FileInfo
	if f, err = d.fakeFileInfo(relPath, objectSize, modTime); err != nil {
		return nil, err
	}

	return f, nil
}

// Return a fakeInfo struct that satisfies the os.FileInfo interface, emulating a file from an S3 object
func (d *S3Driver) fakeFileInfo(name string, size int64, modTime time.Time) (os.FileInfo, error) {
	var err error

	displayPath := name
	// make the path we display relative to the current working directory
	if strings.HasPrefix(displayPath, d.baseDir) {
		if displayPath, err = filepath.Rel(d.baseDir, displayPath); err != nil {
			return nil, err
		}
	}

	isDir := strings.HasSuffix(name, "/")

	mode := os.FileMode(0666)
	if isDir {
		mode = mode | os.ModeDir
	}

	f := fakeInfo{
		name:    displayPath,
		size:    size,
		mode:    mode,
		modTime: modTime,
		isDir:   isDir,
	}

	return f, nil
}

func (d *S3Driver) CanAllocate(cc server.ClientContext, size int) (bool, error) {
	return true, nil
}

func (d *S3Driver) ChmodFile(cc server.ClientContext, path string, mode os.FileMode) error {
	return errors.New("ChmodFile not implemented")
}

func (d *S3Driver) DeleteFile(cc server.ClientContext, path string) error {
	params := &s3.DeleteObjectInput{
		Bucket: &S3_BUCKET_NAME,
		Key:    &path,
	}

	_, err := d.s3Service.DeleteObject(params)
	return err
}

func (d *S3Driver) RenameFile(cc server.ClientContext, from, to string) error {
	// S3 doesn't have rename (or move) so we need to copy (CopyObject) and delete the old one (DeleteObject).
	var err error

	var relFrom, relTo string
	if relFrom, err = filepath.Rel("/", from); err != nil {
		return err
	}

	if relTo, err = filepath.Rel("/", to); err != nil {
		return err
	}

	copySrc := S3_BUCKET_NAME + "/" + relFrom
	copyParams := &s3.CopyObjectInput{
		Bucket:     &S3_BUCKET_NAME,
		Key:        &relTo,
		CopySource: &copySrc,
	}

	if _, err = d.s3Service.CopyObject(copyParams); err != nil {
		return err
	}

	if err = d.DeleteFile(cc, from); err != nil {
		return err
	}

	return nil
}

func (d *S3Driver) GetSettings() *server.Settings {
	config := server.Settings{
		Host:           "0.0.0.0",
		Port:           FTP_PORT,
		MaxConnections: 300,
		MonitorOn:      true,
		MonitorPort:    3379,
	}
	return &config
}

func NewS3Driver() (*S3Driver, error) {

	var err error

	driver := &S3Driver{}

	if driver.s3Session, err = session.NewSession(); err != nil {
		log.Println("error creating S3 session:", err)
		return nil, err
	}

	driver.s3Service = s3.New(driver.s3Session)

	// an empty string corresponds to the root of the S3 bucket
	driver.baseDir = ""

	return driver, nil
}
