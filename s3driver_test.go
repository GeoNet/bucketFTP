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
			var d *S3Driver
			if d, err = NewS3Driver(nil, nil); err != nil {
				t.Error(err)
			}

			if _, err = d.AuthUser(nil, tc.user, tc.passwd); (err != nil) != tc.errExpected {
				t.Error("Expected username/passwd to fail")
			}
		})
	}
}

func TestGetS3Key(t *testing.T) {
	testCases := []struct {
		inputPath, s3Key string
	}{
		{"/", ""},
		{".", ""},
		{"/path", "path"},
		{"/path/", "path/"},
		{"/path//", "path/"},
		{"/path with spaces/", "path with spaces/"},
		{"/nested/path/", "nested/path/"},
	}

	var err error
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.inputPath), func(t *testing.T) {
			var s3Key string
			if s3Key, err = getS3Key(tc.inputPath); err != nil {
				t.Error(err)
			}

			if s3Key != tc.s3Key {
				t.Errorf("expected s3key: '%s' but observed: '%s' from path '%s'", tc.s3Key, s3Key, tc.inputPath)
			}
		})
	}
}
