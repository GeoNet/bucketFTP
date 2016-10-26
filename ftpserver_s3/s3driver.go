package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fclairamb/ftpserver/server"
	"log"
	"os"
	"strings"
)

// SampleDriver defines a very basic serverftp driver
type S3Driver struct {
	baseDir   string
	s3Session *session.Session
	s3Service *s3.S3
}

func (driver *S3Driver) WelcomeUser(cc server.ClientContext) (string, error) {
	cc.SetDebug(true)
	return "Welcome to the GeoNet S3 FTP server", nil
}

func (driver *S3Driver) AuthUser(cc server.ClientContext, user, pass string) (server.ClientHandlingDriver, error) {

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

	return driver, nil
}

func (driver *S3Driver) GetTLSConfig() (*tls.Config, error) {
	return nil, errors.New("TLS not implemented")
}

func (driver *S3Driver) ChangeDirectory(cc server.ClientContext, directory string) error {
	//if directory == "/debug" {
	//	cc.SetDebug(!cc.Debug())
	//	return nil
	//} else if directory == "/virtual" {
	//	return nil
	//}
	//_, err := os.Stat(driver.baseDir + directory)
	//return err
	return errors.New("ChangeDirectory not implemented")
}

func (driver *S3Driver) MakeDirectory(cc server.ClientContext, directory string) error {
	//return os.Mkdir(driver.baseDir+directory, 0777)
	return errors.New("MakeDirectory not implemented")
}

func (driver *S3Driver) ListFiles(cc server.ClientContext) ([]os.FileInfo, error) {

	params := &s3.ListObjectsV2Input{
		Bucket: &S3_BUCKET_NAME,
		//Prefix: &driver.baseDir,
		//Delimiter: keys ending with this string, a bit misleading
	}

	var err error
	var resp *s3.ListObjectsV2Output
	if resp, err = driver.s3Service.ListObjectsV2(params); err != nil {
		return nil, err
	}

	files := []os.FileInfo{}
	for _, f := range resp.Contents {
		fake := fakeInfo{
			name:    *f.Key,
			size:    *f.Size,
			mode:    os.FileMode(0666),
			modTime: *f.LastModified,
			isDir:   strings.HasSuffix(*f.Key, "/"),
		}
		files = append(files, fake)
	}

	return files, nil
}

func (driver *S3Driver) UserLeft(cc server.ClientContext) {

}

func (driver *S3Driver) OpenFile(cc server.ClientContext, path string, flag int) (server.FileStream, error) {
	// our implementation uses an interface that mimics a file but uploads/downloads to S3
	var err error
	var s3file *S3VirtualFile

	if s3file, err = NewS3VirtualFile(path, driver.s3Session, driver.s3Service); err != nil {
		return nil, err
	}

	return s3file, nil
}

func (driver *S3Driver) GetFileInfo(cc server.ClientContext, path string) (os.FileInfo, error) {

	params := &s3.GetObjectInput{
		Bucket: &S3_BUCKET_NAME,
		Key:    &path,
	}

	req, resp := driver.s3Service.GetObjectRequest(params)

	if err := req.Send(); err != nil {
		return nil, err
	}

	f := fakeInfo{
		name:    path,
		size:    *resp.ContentLength,
		mode:    os.FileMode(0666), // No file modes in S3, pretend it's a+rwx.  AWS handles permissions.
		modTime: *resp.LastModified,
		isDir:   false, // TODO: get this
		sys:     nil,   // we don't appear to use this so leave as nil
	}

	return f, nil
}

func (driver *S3Driver) CanAllocate(cc server.ClientContext, size int) (bool, error) {
	return true, nil
}

func (driver *S3Driver) ChmodFile(cc server.ClientContext, path string, mode os.FileMode) error {
	return errors.New("ChmodFile not implemented")
}

func (driver *S3Driver) DeleteFile(cc server.ClientContext, path string) error {
	params := &s3.DeleteObjectInput{
		Bucket: &S3_BUCKET_NAME,
		Key:    &path,
	}

	_, err := driver.s3Service.DeleteObject(params)
	return err
}

func (driver *S3Driver) RenameFile(cc server.ClientContext, from, to string) error {

	// S3 doesn't have rename (or move) so we need to copy (CopyObject) and delete the old one (DeleteObject).  Put off for now.

	//copySrc := S3_BUCKET_NAME + "/" + from
	//copyParams := &s3.CopyObjectInput{
	//	Bucket: &S3_BUCKET_NAME,
	//	Key: &to,
	//	CopySource: &copySrc,
	//}
	//
	//if _, err := driver.s3Service.CopyObject(copyParams); err != nil {
	//	return err
	//}

	return errors.New("RenameFile not implemented")
}

func (driver *S3Driver) GetSettings() *server.Settings {
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

	driver.s3Session, err = session.NewSession()
	if err != nil {
		log.Println("error creating S3 session:", err)
		return nil, err
	}

	driver.s3Service = s3.New(driver.s3Session)

	return driver, nil
}
