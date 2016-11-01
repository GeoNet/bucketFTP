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

type S3Driver struct {
	s3Session *session.Session
	s3Service *s3.S3
	maxKeys   int64
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
	// Query S3 to see if the directory exists.  We return an error if we cannot list it.

	var err error
	var prefix string
	if prefix, err = getS3Key(directory + "/"); err != nil {
		return err
	}

	delimiter := "/"
	params := &s3.ListObjectsV2Input{
		Bucket:    &S3_BUCKET_NAME,
		Prefix:    &prefix,
		Delimiter: &delimiter, // limits the search to directories (they have a trailing slash)
	}

	var resp *s3.ListObjectsV2Output
	if resp, err = d.s3Service.ListObjectsV2(params); err != nil {
		return err
	}

	// prefix of "" is a special case, the root directory of a bucket which can have zero objects
	if prefix == "" {
		return nil
	}

	if *resp.KeyCount == 0 {
		return errors.New("No such directory: " + directory)
	}

	return nil
}

func (d *S3Driver) MakeDirectory(cc server.ClientContext, directory string) error {

	var err error
	var dirname string
	if dirname, err = getS3Key(directory + "/"); err != nil {
		return err
	}

	params := &s3.PutObjectInput{
		Bucket: &S3_BUCKET_NAME,
		Key:    &dirname,
		Body:   bytes.NewReader([]byte("")),
	}

	if _, err = d.s3Service.PutObject(params); err != nil {
		return err
	}

	return nil
}

func (d *S3Driver) ListFiles(cc server.ClientContext) ([]os.FileInfo, error) {

	var err error
	var prefix string
	if prefix, err = getS3Key(cc.Path()); err != nil {
		return nil, err
	}

	// all dirs in S3 apart from root dir should end with /. Non standard between ftp clients.
	if len(prefix) > 0 && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	delimiter := "/" // delimiter keeps the listing from being recursive
	params := &s3.ListObjectsV2Input{
		Bucket:    &S3_BUCKET_NAME,
		Prefix:    &prefix,
		Delimiter: &delimiter,
		MaxKeys:   &d.maxKeys,
	}

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
		if *f.Key == prefix {
			continue
		}

		var fi os.FileInfo
		if fi, err = d.getFakeFileInfo(*f.Key, *f.Size, *f.LastModified); err != nil {
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

func (d *S3Driver) getObjectInfo(key string) (*s3.GetObjectOutput, error) {
	params := &s3.GetObjectInput{
		Bucket: &S3_BUCKET_NAME,
		Key:    &key,
	}

	req, resp := d.s3Service.GetObjectRequest(params)

	if err := req.Send(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (d *S3Driver) GetFileInfo(cc server.ClientContext, path string) (os.FileInfo, error) {

	var err error
	relPath := path
	if relPath, err = getS3Key(path); err != nil {
		return nil, err
	}

	var resp *s3.GetObjectOutput
	// check for directories (trailing slashes) if we can't find the file
	if resp, err = d.getObjectInfo(relPath); err != nil {

		if resp, err = d.getObjectInfo(relPath + "/"); err != nil {
			return nil, err
		} else {
			relPath += "/"
		}
	}

	// resp itself, resp.ContentLength and resp.LastModified are sometimes nil (!) so check for this state.  Aws!
	if resp == nil {
		return nil, fmt.Errorf("Error getting file info for key: %s", relPath)
	}

	var objectSize int64
	if resp.ContentLength != nil {
		objectSize = *resp.ContentLength

		if strings.HasSuffix(relPath, "/") {
			// the size of a directory, just faking it.
			objectSize = 4096
		}
	}

	var modTime time.Time
	if resp.LastModified != nil {
		modTime = *resp.LastModified
	}

	var f os.FileInfo
	if f, err = d.getFakeFileInfo(relPath, objectSize, modTime); err != nil {
		return nil, err
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
	// list objects matching the path, then use DeleteObjects on all of them.  Needed because you must delete all
	// child key/objects belonging to a directory key before deleting that key

	var err error
	var relPath string
	if relPath, err = getS3Key(path); err != nil {
		return err
	}

	var isDir bool
	if isDir, err = d.isS3Dir(relPath); err != nil {
		return err
	}

	if isDir && !strings.HasSuffix(relPath, "/") {
		relPath += "/"
	}

	listParams := &s3.ListObjectsV2Input{
		Bucket:  &S3_BUCKET_NAME,
		Prefix:  &relPath,
		MaxKeys: &d.maxKeys,
		//Delimiter: &delimiter, // empty delim makes this recursive
	}

	var resp *s3.ListObjectsV2Output
	if resp, err = d.s3Service.ListObjectsV2(listParams); err != nil {
		return err
	}

	var delObjects []*s3.ObjectIdentifier
	for _, f := range resp.Contents {
		delObjects = append(delObjects, &s3.ObjectIdentifier{Key: f.Key})
	}

	delParams := &s3.DeleteObjectsInput{
		Bucket: &S3_BUCKET_NAME,
		Delete: &s3.Delete{Objects: delObjects},
	}

	if _, err = d.s3Service.DeleteObjects(delParams); err != nil {
		return err
	}

	return nil
}

// returns true or false depending on if the path exists as a file (no trailing slash) or a directory (trailing slash)
// in S3.
func (d *S3Driver) isS3Dir(path string) (bool, error) {
	var err error
	var s3Path string
	if s3Path, err = getS3Key(path); err != nil {
		return false, err
	}

	_, err = d.getObjectInfo(s3Path)
	if err == nil {
		return false, nil
	}

	_, err = d.getObjectInfo(s3Path + "/")
	if err == nil {
		return true, nil
	}

	return false, errors.New("No such file or directory")
}

func (d *S3Driver) RenameFile(cc server.ClientContext, from, to string) error {
	// S3 doesn't have rename (or move).  We're copying all objects that match the input file or directory key
	// to the new key name

	var err error
	var relFrom, relTo string

	if relFrom, err = getS3Key(from); err != nil {
		return err
	}

	if relTo, err = getS3Key(to); err != nil {
		return err
	}

	var isDir bool
	if isDir, err = d.isS3Dir(relFrom); err != nil {
		return err
	}

	if isDir {
		if !strings.HasSuffix(relFrom, "/") {
			relFrom += "/"
		}
		if !strings.HasSuffix(relTo, "/") {
			relTo += "/"
		}
	}

	listParams := &s3.ListObjectsV2Input{
		Bucket:  &S3_BUCKET_NAME,
		Prefix:  &relFrom,
		MaxKeys: &d.maxKeys,
	}

	var resp *s3.ListObjectsV2Output
	if resp, err = d.s3Service.ListObjectsV2(listParams); err != nil {
		return err
	}

	var srcObjects []*s3.ObjectIdentifier
	for _, f := range resp.Contents {
		srcObjects = append(srcObjects, &s3.ObjectIdentifier{Key: f.Key})
	}

	// copy all destinations objects from source to dest (already ordered from top level directory key)
	for _, objId := range srcObjects {

		toPath := strings.Replace(*objId.Key, relFrom, relTo, 1)
		copySrc := S3_BUCKET_NAME + "/" + *objId.Key
		copyParams := &s3.CopyObjectInput{
			Bucket:     &S3_BUCKET_NAME,
			Key:        &toPath,
			CopySource: &copySrc,
		}

		if _, err = d.s3Service.CopyObject(copyParams); err != nil {
			return err
		}
	}

	// delete original file (or nested directory of matching keys).  Faster than looping over them.
	delParams := &s3.DeleteObjectsInput{
		Bucket: &S3_BUCKET_NAME,
		Delete: &s3.Delete{Objects: srcObjects},
	}

	if _, err = d.s3Service.DeleteObjects(delParams); err != nil {
		return err
	}

	return nil
}

func (d *S3Driver) GetSettings() *server.Settings {
	config := server.Settings{
		Host:           "0.0.0.0",
		Port:           FTP_PORT,
		MaxConnections: 300,
	}
	return &config
}

// Return a fakeInfo struct that satisfies the os.FileInfo interface, emulating a file from an S3 object
func (d *S3Driver) getFakeFileInfo(name string, size int64, modTime time.Time) (os.FileInfo, error) {

	isDir := strings.HasSuffix(name, "/")

	mode := os.FileMode(0666)
	if isDir {
		mode = mode | os.ModeDir
	}

	f := fakeInfo{
		name:    filepath.Base(name),
		size:    size,
		mode:    mode,
		modTime: modTime,
		isDir:   isDir,
	}

	return f, nil
}

func NewS3Driver(s3Session *session.Session, s3Service *s3.S3) (*S3Driver, error) {

	driver := &S3Driver{
		maxKeys: 10000,
		s3Service: s3Service,
		s3Session: s3Session,
	}

	return driver, nil
}

// cleans the input path and queries S3 to see if it's a directory or file key.  Will return an error if it cannot find
// either a file or directory key (with a trailing slash)
func getS3Key(path string) (string, error) {
	var err error
	s3Key := path

	// join paths and keep relative to root dir.
	if strings.HasPrefix(s3Key, "/") {
		if s3Key, err = filepath.Rel("/", path); err != nil {
			return "", err
		}
	}

	// "/" relative to "/" is "." but in S3 this should be ""
	if s3Key == "." {
		s3Key = ""
	}

	// directories in S3 must always have a trailing slash (filepath.Rel removes this)
	if strings.HasSuffix(path, "/") && len(s3Key) > 0 && !strings.HasSuffix(s3Key, "/") {
		s3Key += "/"
	}

	return s3Key, nil
}
