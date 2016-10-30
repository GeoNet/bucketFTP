FROM ubuntu

RUN apt-get update
RUN apt-get install -y ca-certificates
ADD ftpserver_s3 /
#USER nobody
#ENTRYPOINT /ftp-s3
CMD ["/ftpserver_s3"]

# FTP port
EXPOSE 21
