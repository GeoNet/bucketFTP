package main

import (
	"bytes"
	"fmt"
	"github.com/jlaffaye/ftp"
	"gopkg.in/inconshreveable/log15.v2"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
	"github.com/satori/go.uuid"
)

var U string

func init() {
	// disable logging in log15 (used by the ftp server)
	logger := log15.Root()
	logger.SetHandler(log15.DiscardHandler())

	// Use a uuid based on MAC address (should be unique in Travis/Docker)
	U = uuid.NewV1().String()

	// run the main bucketFTP server app in a goroutine
	go main()
}

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

	c.Quit()

	if c, err = getClient(false); err != nil {
		t.Fatal(err)
	}

	err = c.Login("bad_username", "bad_password")
	if err == nil {
		t.Fatal(err)
	}
}

func checkUploadedFile(c *ftp.ServerConn, path string) error {
	// Upload a file to the FTP server using PUT.  GET it back and confirm it's contents

	var err error
	testString := "some text in a file like object"
	data := bytes.NewBufferString(testString)

	if err = c.Stor(path, data); err != nil {
		return err
	}

	var reader io.ReadCloser
	if reader, err = c.Retr(path); err != nil {
		return err
	}
	defer reader.Close()

	var dataRead []byte
	if dataRead, err = ioutil.ReadAll(reader); err != nil {
		return err
	}

	if testString != string(dataRead) {
		return fmt.Errorf("Strings do not match, expected: [%s] but saw [%s]", testString, string(dataRead))
	}

	return nil
}

func TestPutAndGet(t *testing.T) {
	testCases := []struct {
		path        string
		errExpected bool
	}{
		{"/testfile1" + U + ".txt", false},
		{"testfile2" + U + ".txt", false},
		{"file with spaces" + U + ".txt", false},
		{"/invalid_directory" + U + "/testfile3" + U + ".txt", true}, // the parent key does not exist
	}

	var err error
	var c *ftp.ServerConn
	if c, err = getClient(true); err != nil {
		t.Fatal(err)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("putting %s, expecting %v", tc.path, tc.errExpected), func(t *testing.T) {
			if err = checkUploadedFile(c, tc.path); (err != nil) != tc.errExpected {
				t.Error(err)
			}
			defer c.Delete(tc.path)

			if tc.errExpected {
				return
			}

			if err = c.Delete(tc.path); err != nil {
				t.Error(err)
			}
		})
	}

}

func TestDirs(t *testing.T) {
	// test mkdir, cd, pwd, del
	testCases := []struct {
		path        string
		expectedCwd string
		errExpected bool
	}{
		{"/", "/", true}, // shouldn't be able to mkdir /
		{"testdir1" + U, "/testdir1" + U, false},
		{"/testdir2" + U, "/testdir2" + U, false},
		{"/testdir3" + U + "/", "/testdir3" + U + "/", false},
		{"test dir 4" + U, "/test dir 4" + U, false},
		{"testdir5" + U + "/testsubdir" + U, "", true}, // fails, parent key doesn't exist yet
	}

	var err error
	var c *ftp.ServerConn

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s:%v", tc.path, tc.errExpected), func(t *testing.T) {
			if c, err = getClient(true); err != nil {
				t.Fatal(err)
			}
			defer c.Quit()

			if err = c.MakeDir(tc.path); (err != nil) != tc.errExpected {
				t.Fatal(err)
			}

			// all done if we expect the command to fail
			if tc.errExpected {
				return
			}

			if err = c.ChangeDir(tc.path); err != nil {
				t.Error(err)
			}

			var cwd string
			if cwd, err = c.CurrentDir(); err != nil {
				t.Error(err)
			}

			if cwd != tc.expectedCwd {
				t.Errorf("cwd [%s] differs from expected path [%s]", cwd, tc.expectedCwd)
			}

			// test getting and putting a file from this directory
			testFile := "testfile" + U
			if err = checkUploadedFile(c, testFile); err != nil {
				t.Error(err)
			}
			defer c.Delete(testFile)

			// special case, we're done testing the root dir
			if tc.path == "/" {
				return
			}

			if err = c.ChangeDir("/"); err != nil {
				t.Error(err)
			}

			if err = c.Delete(tc.path); err != nil {
				t.Error(err)
			}

			// directory doesn't exist any more, err should be non nil
			if err = c.ChangeDir(tc.path); err == nil {
				t.Error("expected ChangeDir to fail but it worked")
			}

		})
	}
}

func TestRename(t *testing.T) {
	// Test that renaming a file or directory works
	testCases := []struct {
		oldPath     string
		mkDirs      []string
		newPath     string
		isDir       bool
		errExpected bool
	}{
		// dirs
		{"/", []string{}, "/", true, true}, // shouldn't be able to cp a dir to itself
		{"testdir1" + U, []string{}, "newtestdir1" + U, true, false},
		{"/testdir2" + U, []string{}, "/newtestdir2" + U, true, false},
		{"/testdir3" + U + "/", []string{}, "/testdir4/subdir" + U, true, true}, // parent key doesn't exist, should fail
		// files
		{"newfile1" + U, []string{}, "newfile2" + U, false, false},
		{"/newfile3" + U, []string{}, "newfile4" + U, false, false},
		{"/newfile3" + U, []string{}, "new file 4" + U, false, false},
		{"/dir with spaces" + U + "/testfile1" + U + ".txt", []string{"dir with spaces" + U},
			"new dir 4" + U + "/more spaces1" + U + ".txt", false, true}, // parent dir does not exist
		{"/dir with spaces" + U + "/testfile2" + U + ".txt", []string{"dir with spaces" + U, "new dir 5" + U},
			"new dir 5" + U + "/more spaces2" + U + ".txt", false, false}, // parent dir exists
	}

	var err error
	var c *ftp.ServerConn
	testString := "some more text in a file like object"

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s:%s,%v", tc.oldPath, tc.newPath, tc.errExpected), func(t *testing.T) {
			if c, err = getClient(true); err != nil {
				t.Fatal(err)
			}
			defer c.Quit()

			// create any dirs required
			if err = mkDirs(c, tc.mkDirs); err != nil {
				t.Error(err)
			}

			if tc.isDir {
				// don't need to mkdir on /
				if tc.oldPath != "/" {
					if err = c.MakeDir(tc.oldPath); err != nil {
						t.Error(err)
					}
				}

				if err = c.Rename(tc.oldPath, tc.newPath); (err != nil) != tc.errExpected {
					t.Fatal(err)
				}

				// call had expected error so bail out
				if tc.errExpected {
					_ = c.Delete(tc.oldPath)
					return
				}

				if err = c.ChangeDir(tc.newPath); err != nil {
					t.Error(err)
				}

				// shouldn't work except on /
				err = c.ChangeDir(tc.oldPath)
				if tc.oldPath != "/" && err == nil {
					t.Errorf("ChangeDir was expected to fail when cd-ing to: %s", tc.oldPath)
				}
			} else {

				// write some info to the test file if we're renaming it
				data := bytes.NewBufferString(testString)

				if err = c.Stor(tc.oldPath, data); err != nil {
					t.Error(err)
				}

				// move the file
				if err = c.Rename(tc.oldPath, tc.newPath); (err != nil) != tc.errExpected {
					t.Fatal(err)
				}

				if tc.errExpected {
					return
				}

				var reader io.ReadCloser
				if reader, err = c.Retr(tc.newPath); err != nil {
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

				//// TODO: the ftp client doesn't observe the error.  Should fix this.  Manual tests show the file is deleted.
				//if _, err = c.Retr(tc.oldPath); err == nil {
				//	t.Errorf("file should not exist: %s", tc.oldPath)
				//}

			}

			if err = c.ChangeDir("/"); err != nil {
				t.Error(err)
			}

			if err = c.Delete(tc.newPath); (err != nil) != tc.errExpected {
				t.Fatal(err)
			}

			if err = rmDirs(c, tc.mkDirs); err != nil {
				t.Error(err)
			}

		})
	}
}

func TestDirRenameDelete(t *testing.T) {
	// Test that renaming a directory containing file(s) works as expected
	var err error
	var c *ftp.ServerConn
	if c, err = getClient(true); err != nil {
		t.Fatal(err)
	}

	dirA := "/dirA" + U
	dirB := "/dirB" + U
	newDirA := "/dirB" + U + "/subdir" + U
	if err = c.MakeDir(dirA); err != nil {
		t.Error(err)
	}

	if err = c.MakeDir(dirB); err != nil {
		t.Error(err)
	}

	if err = c.ChangeDir(dirA); err != nil {
		t.Error(err)
	}

	testFile := "testfile" + U
	if err = checkUploadedFile(c, testFile); err != nil {
		t.Error(err)
	}
	defer c.Delete(testFile)

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

func TestListFiles(t *testing.T) {
	// check the list of files (not their contents)
	startDir := "/listfiles" + U
	files := []string{"file1" + U, "file2" + U, "file3" + U}
	dirs := []string{"subdir1" + U, "subdir2" + U, "subdir2" + U + "/deepdir" + U}

	var err error
	var c *ftp.ServerConn
	if c, err = getClient(true); err != nil {
		t.Fatal(err)
	}
	defer c.Quit()

	if err = c.MakeDir(startDir); err != nil {
		t.Fatal(err)
	}

	if err = c.ChangeDir(startDir); err != nil {
		t.Fatal(err)
	}

	testString := "some text in a file like object"
	data := bytes.NewBufferString(testString)

	// store a bunch of files in the starting dir
	for _, f := range files {

		if err = c.Stor(f, data); err != nil {
			t.Fatal(err)
		}
	}

	// also store them in the subdirs
	for _, dir := range dirs {
		if err = c.ChangeDir(startDir); err != nil {
			t.Fatal(err)
		}

		if err = c.MakeDir(dir); err != nil {
			t.Fatal(err)
		}

		if err = c.ChangeDir(dir); err != nil {
			t.Fatal(err)
		}

		for _, f := range files {

			if err = c.Stor(f, data); err != nil {
				t.Fatal(err)
			}
		}
	}

	// check the file listing is complete but not recursive
	var entries []*ftp.Entry
	if entries, err = c.List(startDir); err != nil {
		t.Error(err)
	}

	// Arg, this ftp client doesn't list directories.  At least we know the listing isn't recursive.
	for i, e := range entries {
		if e.Name != files[i] {
			t.Errorf("Expected file name '%s' but observed '%s'", e.Name, files[i])
		}
	}

	if err = c.ChangeDir("/"); err != nil {
		t.Fatal(err)
	}

	if err = c.Delete(startDir); err != nil {
		t.Fatal(err)
	}
}

func mkDirs(c *ftp.ServerConn, dirs []string) error {
	var err error
	for _, d := range dirs {
		if len(d) < 1 {
			continue
		}

		if err = c.MakeDir(d); err != nil {
			return err
		}
	}

	return nil
}

func rmDirs(c *ftp.ServerConn, dirs []string) error {
	var err error
	for _, d := range dirs {
		if len(d) < 1 {
			continue
		}

		if err = c.Delete(d); err != nil {
			return err
		}
	}

	return nil
}
