package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

// testing several things that don't depend on S3 (auth, etc)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestAuthUser(t *testing.T) {
	testCases := []struct {
		user, passwd string
		errExpected  bool
	}{
		{os.Getenv("FTP_USER"), os.Getenv("FTP_PASSWD"), false},
		{"", "", true},
		{"invalid", "", true},
		{"invalid", "badpasswd", true},
		{"", "badpasswd", true},
	}

	var err error
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s, %s, %t", tc.user, tc.passwd, tc.errExpected), func(t *testing.T) {
			d := &S3Driver{ftpUser: os.Getenv("FTP_USER"), ftpPasswd: os.Getenv("FTP_PASSWD")}
			if _, err = d.AuthUser(nil, tc.user, tc.passwd); (err != nil) != tc.errExpected {
				t.Errorf("Expected username/passwd to fail: %s: %s", tc.user, tc.passwd)
			}
		})
	}
}

func TestGetS3Key(t *testing.T) {
	testCases := []struct {
		rootPrefix, inputPath, s3Key string
	}{
		{"", "", ""},
		{"", "/", ""},
		{"", ".", ""},
		{"", "./", ""},
		{"", "/path", "path"},
		{"", "/path/", "path/"},
		{"", "/path//", "path/"},
		{"", "/path with spaces/", "path with spaces/"},
		{"", "/nested/path/", "nested/path/"},
		// The same tests with a rootPrefix:
		{"testprefix/", "", "testprefix/"},
		{"testprefix/", "/", "testprefix/"},
		{"testprefix/", ".", "testprefix/"},
		{"testprefix/", "/path", "testprefix/path"},
		{"testprefix/", "/path/", "testprefix/path/"},
		{"testprefix/", "/path//", "testprefix/path/"},
		{"testprefix/", "/path with spaces/", "testprefix/path with spaces/"},
		{"testprefix/", "/nested/path/", "testprefix/nested/path/"},
	}

	var err error
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.inputPath), func(t *testing.T) {
			sd := S3Driver{rootPrefix: tc.rootPrefix}
			var s3Key string
			if s3Key, err = sd.getS3Key(tc.inputPath); err != nil {
				t.Error(err)
			}

			if s3Key != tc.s3Key {
				t.Errorf("expected s3key: '%s' but observed: '%s' from path '%s'", tc.s3Key, s3Key, tc.inputPath)
			}
		})
	}
}
