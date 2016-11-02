FROM scratch
ADD ./ /
#USER nobody
CMD ["/bucketFTP"]
# FTP port
EXPOSE 21
