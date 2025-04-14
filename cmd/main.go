package main

import (
	"log"
	"os"

	"github.com/eyad-hussein/sync_with_cloud/internal"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalln(err)
	}

	opts := internal.NewUploaderOpts([]string{""}, []string{""}, *internal.NewAuthClient(os.Getenv("SERVICE_ACCOUNT_CREDENTIALS_PATH")))

	uploader := internal.NewUploader(opts)

	if err := uploader.Upload(); err != nil {
		log.Fatalln(err)
	}
}
