package internal

import (
	"context"
	"fmt"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type UploaderOpts struct {
	localPaths  []string
	remotePaths []string
	authClient  *AuthClient
}

func NewUploaderOpts(localPaths, remotePaths []string, authClient AuthClient) UploaderOpts {
	return UploaderOpts{
		localPaths:  localPaths,
		remotePaths: remotePaths,
		authClient:  &authClient,
	}
}

type Uploader struct {
	opts *UploaderOpts
}

func NewUploader(opts UploaderOpts) *Uploader {
	return &Uploader{opts: &opts}
}

func (u *Uploader) Upload() error {
	auth, err := u.opts.authClient.GetAuthClient()

	if err != nil {
		return err
	}

	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(auth))
	if err != nil {
		return err
	}

	query := "mimeType='application/vnd.google-apps.folder' and trashed=false"

	r, err := srv.Files.List().
		Q(query).
		PageSize(100). // Adjust page size as needed
		Fields("nextPageToken, files(id, name)").
		Do()
	if err != nil {
		return err
	}

	if len(r.Files) == 0 {
		fmt.Println("No files found")
	}

	for _, i := range r.Files {
		fmt.Println(i.Name + " " + i.Id)
	}

	return nil
}
