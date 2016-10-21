# FTP Server for S3

This is a Go implementation of an FTP server to allow a user to put or 
get files from a single S3 bucket.

## Security

FTP is an extremely insecure protocol.  We use scientific instruments that
only support FTP, otherwise would avoid it.  We strongly recommend using
a private network when using FTP.

User auth is extremely simple.  It's checking against environment variables.
This will likely be changed.

TLS is not currently supported.

## Implementation

This utility uses the ftpserver package from github.com/fclairamb/ftpserver/server 
with a driver that uses the AWS SDK for Go: github.com/aws/aws-sdk-go/aws.

The environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY should
be supplied (available from the AWS console).

## Important Notes

* Active FTP transfers are not supported, only passive FTP.
* No buffering or saving to tempfiles is done on the FTP server, this 
should let a user upload or download large files (untested).
* This is a minimal implementation, only the required commands have been
implemented: get, put.
* It was intended to run this from Docker but this server uses random
port numbers which are poorly supported by docker.
