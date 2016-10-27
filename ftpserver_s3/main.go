package main

import (
	"github.com/fclairamb/ftpserver/server"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var (
	ftpServer      *server.FtpServer
	FTP_PORT_STR   = os.Getenv("FTP_PORT")
	FTP_PORT       int
	S3_BUCKET_NAME = os.Getenv("S3_BUCKET_NAME")
	FTP_USER       = os.Getenv("FTP_USER")
	FTP_PASSWD     = os.Getenv("FTP_PASSWD")
)

func init() {
	switch "" {
	case FTP_PORT_STR:
		log.Fatal("Error: environment variable FTP_PORT is not set")
	case S3_BUCKET_NAME:
		log.Fatal("Error: environment variable S3_BUCKET_NAME is not set")
	case FTP_USER:
		log.Fatal("Error: environment variable FTP_USER is not set")
	case FTP_PASSWD:
		log.Fatal("Error: environment variable FTP_PASSWD is not set")
	}

	var err error
	if FTP_PORT, err = strconv.Atoi(FTP_PORT_STR); err != nil {
		log.Fatal("Error parsing FTP_PORT as an integer", err)
	}
}

func main() {
	var err error
	var driver *S3Driver
	if driver, err = NewS3Driver(); err != nil {
		log.Fatal("error creating S3 FTP driver", err)
	}

	ftpServer = server.NewFtpServer(driver)

	go signalHandler()

	if err = ftpServer.ListenAndServe(); err != nil {
		log.Println("Problem listening", err)
	}
}

func signalHandler() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM)
	for {
		switch <-ch {
		case syscall.SIGTERM:
			ftpServer.Stop()
			break
		}
	}
}
