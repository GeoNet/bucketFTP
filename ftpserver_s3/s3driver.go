package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fclairamb/ftpserver/server"
	"log"
	"os"
)

// SampleDriver defines a very basic serverftp driver
type SampleDriver struct {
	//baseDir   string
	//tlsConfig *tls.Config
}

func (driver *SampleDriver) WelcomeUser(cc server.ClientContext) (string, error) {
	cc.SetDebug(true)
	return "Welcome to the GeoNet S3 FTP server", nil
}

func (driver *SampleDriver) AuthUser(cc server.ClientContext, user, pass string) (server.ClientHandlingDriver, error) {

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

func (driver *SampleDriver) GetTLSConfig() (*tls.Config, error) {
	return nil, errors.New("TLS not implemented")
}

func (driver *SampleDriver) ChangeDirectory(cc server.ClientContext, directory string) error {
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

func (driver *SampleDriver) MakeDirectory(cc server.ClientContext, directory string) error {
	//return os.Mkdir(driver.baseDir+directory, 0777)
	return errors.New("MakeDirectory not implemented")
}

func (driver *SampleDriver) ListFiles(cc server.ClientContext) ([]os.FileInfo, error) {

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

func (driver *SampleDriver) UserLeft(cc server.ClientContext) {

}

func (driver *SampleDriver) OpenFile(cc server.ClientContext, path string, flag int) (server.FileStream, error) {
	// our implementation uses an interface that mimics a file but uploads/downloads to S3
	var err error
	var s3file *S3VirtualFile

	if s3file, err = NewS3VirtualFile(path); err != nil {
		return nil, err
	}

	return s3file, nil
}

func (driver *SampleDriver) GetFileInfo(cc server.ClientContext, path string) (os.FileInfo, error) {
	//path = driver.baseDir + path

	return nil, errors.New("GetFileInfo not implemented")
}

func (driver *SampleDriver) CanAllocate(cc server.ClientContext, size int) (bool, error) {
	return true, nil
}

func (driver *SampleDriver) ChmodFile(cc server.ClientContext, path string, mode os.FileMode) error {
	//path = driver.baseDir + path
	//return os.Chmod(path, mode)
	return errors.New("ChmodFile not implemented")
}

func (driver *SampleDriver) DeleteFile(cc server.ClientContext, path string) error {
	//path = driver.baseDir + path
	//
	//return os.Remove(path)
	return errors.New("DeleteFile not implemented")
}

func (driver *SampleDriver) RenameFile(cc server.ClientContext, from, to string) error {
	//from = driver.baseDir + from
	//to = driver.baseDir + to
	//
	//return os.Rename(from, to)
	return errors.New("RenameFile not implemented")
}

func (driver *SampleDriver) GetSettings() *server.Settings {
	config := server.Settings{
		Host:           "0.0.0.0",
		Port:           FTP_PORT,
		MaxConnections: 300,
		MonitorOn:      true,
		MonitorPort:    3379,
	}
	return &config
}

// Note: This is not a mistake. Interface can be pointers. There seems to be a lot of confusion around this in the
//       server_ftp original code.
func S3Driver() *SampleDriver {
	//dir, err := ioutil.TempDir("", "ftpserver")
	//if err != nil {
	//	log15.Error("Could not find a temporary dir", "err", err)
	//}

	driver := &SampleDriver{}
	//os.MkdirAll(driver.baseDir, 0777)
	return driver
}

//type VirtualFileInfo struct {
//	name string
//	size int64
//	mode os.FileMode
//}
//
//func (f VirtualFileInfo) Name() string {
//	return f.name
//}
//
//func (f VirtualFileInfo) Size() int64 {
//	return f.size
//}
//
//func (f VirtualFileInfo) Mode() os.FileMode {
//	return f.mode
//}
//
//func (f VirtualFileInfo) IsDir() bool {
//	return f.mode.IsDir()
//}
//
//func (f VirtualFileInfo) ModTime() time.Time {
//	return time.Now().UTC()
//}
//
//func (f VirtualFileInfo) Sys() interface{} {
//	return nil
//}
