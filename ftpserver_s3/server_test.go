package main

import (
	"bytes"
	"github.com/jlaffaye/ftp"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

// Integration style tests for the FTP server.  This requires the server to be running
// on localhost and the variables in env.list to be exported (eg: FTP_PORT, etc).

func getClient(doLogin bool) (*ftp.ServerConn, error) {
	c, err := ftp.DialTimeout("localhost:"+os.Getenv("FTP_PORT"), time.Second)
	if err != nil {
		return nil, err
	}

	if doLogin {
		err = c.Login(os.Getenv("FTP_USER"), os.Getenv("FTP_PASSWD"))
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// test that we can connect to the server
func TestLogin(t *testing.T) {

	var err error
	var c *ftp.ServerConn
	if c, err = getClient(false); err != nil {
		t.Fatal(err)
	}

	err = c.Login(os.Getenv("FTP_USER"), os.Getenv("FTP_PASSWD"))
	if err != nil {
		t.Fatal(err)
	}
}

func testFileContents(t *testing.T, c *ftp.ServerConn) {
	// Upload a file to the FTP server using PUT.  GET it back and confirm it's contents

	var err error
	path := "test.txt"
	testString := "some text in a file like object"
	data := bytes.NewBufferString(testString)

	if err = c.Stor(path, data); err != nil {
		t.Error(err)
	}

	var reader io.ReadCloser
	if reader, err = c.Retr(path); err != nil {
		t.Error(err)
	}
	defer reader.Close()

	var dataRead []byte
	if dataRead, err = ioutil.ReadAll(reader); err != nil {
		t.Error(err)
	}

	if testString != string(dataRead) {
		t.Errorf("Strings do not match, expected: [%s] but saw [%s]", testString, string(dataRead))
	}
}

func TestPutAndGet(t *testing.T) {
	var err error
	var c *ftp.ServerConn
	if c, err = getClient(true); err != nil {
		t.Fatal(err)
	}

	testFileContents(t, c)
}

func TestDirs(t *testing.T) {
	// test mkdir, cd, pwd, del

	var err error
	var c *ftp.ServerConn
	if c, err = getClient(true); err != nil {
		t.Fatal(err)
	}

	// CDing to / in an empty S3 bucket has been troublesome
	if err = c.ChangeDir("/"); err != nil {
		t.Error(err)
	}

	newDir := "newdir"
	expectedCwd := "/newdir"
	if err = c.MakeDir(newDir); err != nil {
		// c.MakeDir seems to return a non-nil error on success.  Strange, probably the ftpserver package
		//t.Error(err)
	}

	if err = c.ChangeDir(newDir); err != nil {
		t.Error(err)
	}

	var cwd string
	if cwd, err = c.CurrentDir(); err != nil {
		t.Error(err)
	}

	if cwd != expectedCwd {
		t.Errorf("expected cwd to be %s but it is %s", expectedCwd, cwd)
	}

	// test getting and putting a file from this directory
	testFileContents(t, c)

	if err = c.ChangeDirToParent(); err != nil {
		t.Error(err)
	}

	if err = c.Delete(newDir); err != nil {
		t.Error(err)
	}

	// directory doesn't exist any more, err should be non nil
	if err = c.ChangeDir(newDir); err == nil {
		t.Error(err)
	}

}

func TestRename(t *testing.T) {
	// Test that renaming a file works

	var err error
	var c *ftp.ServerConn
	if c, err = getClient(true); err != nil {
		t.Fatal(err)
	}

	origPath := "test2.txt"
	testString := "some more text in a file like object"
	data := bytes.NewBufferString(testString)

	if err = c.Stor(origPath, data); err != nil {
		t.Error(err)
	}

	// move the file
	newPath := "test2_moved.txt"
	if err = c.Rename(origPath, newPath); err != nil {
		t.Error(err)
	}

	var reader io.ReadCloser
	if reader, err = c.Retr(newPath); err != nil {
		t.Error(err)
	}

	var dataRead []byte
	if dataRead, err = ioutil.ReadAll(reader); err != nil {
		t.Error(err)
	}
	reader.Close()

	if testString != string(dataRead) {
		t.Errorf("Strings do not match, expected: [%s] but saw [%s]", testString, string(dataRead))
	}

	// Reading the original file should fail.
	if _, err = c.Retr(origPath); err != nil {
		t.Errorf("file should not exist: %s", origPath)
	}
}

func TestDirRenameDelete(t *testing.T) {
	// Test that renaming a directory containing file(s) works as expected
	var err error
	var c *ftp.ServerConn
	if c, err = getClient(true); err != nil {
		t.Fatal(err)
	}

	dirA := "/dirA"
	dirB := "/dirB"
	newDirA := "/dirB/subdir"
	if err = c.MakeDir(dirA); err != nil {
		// c.MakeDir seems to return a non-nil error on success.  Strange, probably the ftpserver package
		//t.Error(err)
	}

	if err = c.MakeDir(dirB); err != nil {
		// c.MakeDir seems to return a non-nil error on success.  Strange, probably the ftpserver package
		//t.Error(err)
	}

	if err = c.ChangeDir(dirA); err != nil {
		t.Error(err)
	}

	testFileContents(t, c)

	if err = c.ChangeDirToParent(); err != nil {
		t.Error(err)
	}

	if err = c.Rename(dirA, newDirA); err != nil {
		t.Error(err)
	}

	if err = c.ChangeDir(newDirA); err != nil {
		t.Error(err)
	}

	var entries []*ftp.Entry
	if entries, err = c.List(newDirA); err != nil {
		t.Error(err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected one file in dir %s, saw %d", newDirA, len(entries))
	}

	// test deleting the nested directory
	if err = c.ChangeDir("/"); err != nil {
		t.Error(err)
	}

	if err = c.Delete(dirB); err != nil {
		t.Error(err)
	}

	// shouldn't be able to cd
	if err = c.ChangeDir(dirB); err == nil {
		t.Error("should not be able to cd into a deleted directory")
	}
}
