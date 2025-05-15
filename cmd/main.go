package main

import (
	"log"
	"os"

	"github.com/eyad-hussein/sync_with_cloud/internal"
	"gopkg.in/yaml.v3"
)

func main() {

	cgfFile, err := os.ReadFile("drive-sync.yaml")
	if err != nil {
		log.Fatalln(err)
	}

	var cfg internal.Config
	if err := yaml.Unmarshal(cgfFile, &cfg); err != nil {
		log.Fatalln(err)
	}

	opts := internal.NewUploaderOpts(
		cfg,
		*internal.NewAuthClient(cfg.CredentialsFile),
	)

	uploader := internal.NewUploader(opts)

	if err := uploader.InitUploader(); err != nil {
		log.Fatalln(err)
	}

	if err := uploader.Upload(); err != nil {
		log.Fatalln(err)
	}
}
