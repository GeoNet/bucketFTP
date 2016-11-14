FROM scratch
COPY . /
CMD ["/bucketFTP"]
# FTP port
EXPOSE 21
