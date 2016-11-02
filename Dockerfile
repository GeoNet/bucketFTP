FROM scratch
ADD ./ /
#USER nobody
CMD ["/ftpserver_s3"]
# FTP port
EXPOSE 21
