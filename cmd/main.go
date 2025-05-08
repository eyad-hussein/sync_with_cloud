package main

import (
	"log"
	"os"

	"github.com/eyad-hussein/sync_with_cloud/internal"
	"github.com/joho/godotenv"
)

func main() {

	// f, err := os.Open("/media/watashi-2/watashi-ubuntu")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer f.Close()

	// ff, err := f.Stat()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(ff.IsDir())
	// return
	if err := godotenv.Load(); err != nil {
		log.Fatalln(err)
	}

	serviceAccountCredentialsPath := os.Getenv("SERVICE_ACCOUNT_CREDENTIALS_PATH")
	remoteRootFolderID := os.Getenv("REMOTE_ROOT_FOLDER_ID")

	opts := internal.NewUploaderOpts(
		remoteRootFolderID,
		map[string]string{
			"/media/watashi-2/watashi-ubuntu": "watashi-ubuntu",
			"/media/watashi-2/text.txt":       "watashi-ubuntu/not-found-dir1/not-found-dir2/not-found",
		},
		*internal.NewAuthClient(serviceAccountCredentialsPath),
	)

	uploader := internal.NewUploader(opts)

	if err := uploader.InitUploader(); err != nil {
		log.Fatalln(err)
	}

	if err := uploader.Upload(); err != nil {
		log.Fatalln(err)
	}
}
