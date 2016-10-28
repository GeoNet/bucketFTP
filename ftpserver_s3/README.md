# FTP Server for S3

This is a Go implementation of an FTP server which uses Amazon S3 for storage
of file.  It's easy to build and supports basic FTP commands such as get, put, 
delete, ls, cd, rename, and mkdir.  It is free and open source, using the 
Apache 2.0 license.

## Security

FTP is an extremely insecure protocol.  We use hardware that requires 
an FTP server for transferring files.  We strongly recommend using a 
private network when using FTP.

User auth is extremely simple.  It's checking against environment variables.
This will likely change in the future.

TLS is not currently implemented but is supported by the ftp server.

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
* You can then use the FTP client (eg: command line ftp or FileZilla) to 
put/get/cd/mkdir/rename/del files and directories on S3.

## Important Notes

* Active FTP transfers are not supported, only passive FTP.
* No buffering or saving to temp files is done on the FTP server, this 
should let a user upload or download large files.
* This is a minimal implementation, only the required FTP commands have been
implemented: get, put, delete, ls, cd, rename, mkdir.
* It was intended to run this from Docker but this server uses random
port numbers which are difficult to support.
* All dependencies are vendored using govendor.  Recent versions of Go
should automatically use these packages making it easy to build.
* Globbing of files (eg: *.jpg) is not supported
