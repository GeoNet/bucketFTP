# FTP Server for S3

This is a Go implementation of an FTP server to allow a user to put or 
get files from a single S3 bucket.  It is free and open source, using 
the Apache 2.0 license.

## Security

FTP is an extremely insecure protocol.  We use equipment that requires 
an FTP server for transferring files.  We strongly recommend using a 
private network when using FTP.

User auth is extremely simple.  It's checking against environment variables.
This will likely be changed.

TLS is not currently implemented.

File modes on S3 are faked.  Attempting to read or modify a file on S3 with
insufficient permissions will raise an error.

## Implementation

This utility uses the [ftpserver] (github.com/fclairamb/ftpserver/server) package. 
with a custom driver that uses the [AWS SDK for Go] (github.com/aws/aws-sdk-go/aws).

The environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY should
be supplied (These are available from the AWS console for S3).

## Quickstart

* clone this git repo (or download the archive) from github.com
* cd to the ftpserver_s3 directory
* run the command `go build`
* open the file env.list.  Export each environment variable listed with the appropriate
value.  Eg: `export FTP_PORT=3000`.  You will need valid AWS credentials.
* run the ftpserver: `./ftpserver_s3`
* use an FTP client to connect to your running server, eg on Linux connect to the ftp
server running on localhost at port 3000: `ftp -p localhost 3000`

## Important Notes

* Active FTP transfers are not supported, only passive FTP.
* No buffering or saving to temp files is done on the FTP server, this 
should let a user upload or download large files (untested).
* This is a minimal implementation, only the required commands have been
implemented: get, put, delete, ls, cd, rename, mkdir.
* It was intended to run this from Docker but this server uses random
port numbers which are difficult to support in Docker.
* All dependencies are vendored using govendor.  Recent versions of Go
should automatically use these packages.
* Globbing (eg: *.jpg) is not supported
