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

	serviceAccountCredentialsPath := os.Getenv("SERVICE_ACCOUNT_CREDENTIALS_PATH")
	remoteRootFolderID := os.Getenv("REMOTE_ROOT_FOLDER_ID")

	opts := internal.NewUploaderOpts(
		remoteRootFolderID,
		map[string]string{
			"/media/watashi-2/watashi-ubuntu/text.txt": "watashi-ubuntu/text.txt",
			"/media/watashi-2/watashi-ubuntu":          "watashi-ubuntu",
			"/media/watashi-2":                         "not-found",
		},
		*internal.NewAuthClient(serviceAccountCredentialsPath),
	)

	uploader := internal.NewUploader(opts)

	if err := uploader.Upload(); err != nil {
		log.Fatalln(err)
	}
}
