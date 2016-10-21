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

	driver.baseDir = "/"

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

	//if cc.Path() == "/virtual" {
	//	files := make([]os.FileInfo, 0)
	//	files = append(files,
	//		VirtualFileInfo{
	//			name: "localpath.txt",
	//			mode: os.FileMode(0666),
	//			size: 1024,
	//		},
	//		VirtualFileInfo{
	//			name: "file2.txt",
	//			mode: os.FileMode(0666),
	//			size: 2048,
	//		},
	//	)
	//	return files, nil
	//}
	//
	//path := driver.baseDir + cc.Path()
	//
	//files, err := ioutil.ReadDir(path)
	//
	//// We add a virtual dir
	//if cc.Path() == "/" && err == nil {
	//	files = append(files, VirtualFileInfo{
	//		name: "virtual",
	//		mode: os.FileMode(0666) | os.ModeDir,
	//		size: 4096,
	//	})
	//}
	//
	//return files, err
	log.Println("Hello listing some files...")
	return nil, errors.New("ListFiles not implemented")
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
	//path = driver.baseDir + path

	return nil, errors.New("GetFileInfo not implemented")
}

func (driver *S3Driver) CanAllocate(cc server.ClientContext, size int) (bool, error) {
	return true, nil
}

func (driver *S3Driver) ChmodFile(cc server.ClientContext, path string, mode os.FileMode) error {
	//path = driver.baseDir + path
	//return os.Chmod(path, mode)
	return errors.New("ChmodFile not implemented")
}

func (driver *S3Driver) DeleteFile(cc server.ClientContext, path string) error {
	//path = driver.baseDir + path
	//
	//return os.Remove(path)
	return errors.New("DeleteFile not implemented")
}

func (driver *S3Driver) RenameFile(cc server.ClientContext, from, to string) error {
	//from = driver.baseDir + from
	//to = driver.baseDir + to
	//
	//return os.Rename(from, to)
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
