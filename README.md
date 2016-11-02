# FTP Server for S3

This is a Go implementation of an FTP server which uses Amazon S3 for storage
of files.  It's easy to build and supports basic FTP commands such as get, put, 
delete, ls, cd, rename, and mkdir.  It is free and open source, using the 
Apache 2.0 license.

## Security

FTP is an insecure protocol.  We use specialised hardware that only supports 
FTP so are forced to use it.  We strongly recommend using a private network 
when using FTP.

User auth is extremely simple.  It's checking against environment variables.
This will likely change in the future.  One option would be to use the S3 
credentials for the username/password, but we didn't want these accidentally 
transmitted over the internet in plain text.

TLS is not currently implemented but is supported by the upstream ftp server.

File modes on S3 are faked.  Attempting to read or modify a file on S3 with
insufficient permissions will raise an error.

## Implementation

This utility uses the [ftpserver] (github.com/fclairamb/ftpserver/server) package. 
with a custom driver that uses the [AWS SDK for Go] (github.com/aws/aws-sdk-go/aws).

The environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY should
be set when running the FTP server (These are available from the AWS console for S3).

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

## Building and running from Docker

* Install and test Docker on your system.
* build the docker container using the script (on systems supporting bash): `./build.sh`.  This
builds in the Alpine Go container and creates a new scratch based container containing the FTP 
server executable and ssl certs required by the AWS SDK.
* it should report something similar to "Successfully built b5245065b234"
* run the container with the command `docker run -p21:21 --env-file env.list -it b5245065b234`, 
using the name of the container you just built as the last argument.  This will run the server
in a terminal with stderr/stdout being printed to the screen.
* The only exposed port is port 21 (the default FTP port).  All connections must be in passive mode.
* This container can be pushed to any docker repo or run from Amazon's container service or any
other cloud service that runs docker containers.  Managing config as environment variables keeps 
this docker friendly.

## Important Notes

* Active FTP transfers are not supported, only passive FTP.
* No buffering or saving to temp files is done on the FTP server, this 
should let a user upload or download large files.
* This is a minimal implementation, only the required FTP commands have been
implemented: get, put, delete, ls, cd, rename, mkdir.
* All dependencies are vendored using govendor.  Recent versions of Go
should automatically use these packages making it easy to build.
* Globbing of files (eg: *.jpg) is not supported.
* Symbolic links are unsupported and will be ignored.
* AWS limits the number of objects returned in certain operations such as 
ListObjectsV2.  The limit is currently hardcoded to 10000.  This will cause
problems if you exceed this limit.
