FROM scratch
ADD bucketFTP /
CMD ["/bucketFTP"]
# FTP port
EXPOSE 21
