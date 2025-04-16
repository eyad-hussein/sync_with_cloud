package internal

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type UploaderOpts struct {
	remoteRootFolderID string
	pathsToUpdate      map[string]string
	authClient         *AuthClient
}

func NewUploaderOpts(remoteRootFolderID string, pathsToUpdate map[string]string, authClient AuthClient) UploaderOpts {
	return UploaderOpts{
		remoteRootFolderID: remoteRootFolderID,
		pathsToUpdate:      pathsToUpdate,
		authClient:         &authClient,
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

	var currentParentID string
	for key, val := range u.opts.pathsToUpdate {
		fmt.Println(key)

		pathSegments := strings.Split(val, "/")

		currentParentID = u.opts.remoteRootFolderID

		for _, segment := range pathSegments {

			fmt.Println(segment)
			query := fmt.Sprintf("name = '%s' and (mimeType = 'application/vnd.google-apps.folder' or mimeType != 'application/vnd.google-apps.folder') and '%s' in parents", segment, currentParentID)

			fmt.Println(query)
			r, err := srv.Files.List().Q(query).Do()

			if err != nil {
				return err
			}

			if len(r.Files) == 0 {
				return fmt.Errorf("No folders")
			}

			currentParentID = r.Files[0].Id
			fmt.Printf("Found folder '%s' with ID '%s' Count: %d Kind: %s\n,", r.Files[0].Name, r.Files[0].Id, len(r.Files), r.Files[0].MimeType)
		}
	}

	return nil
}

func (u *Uploader) updateFile() {

}

func (u *Uploader) updateFolder() {

}
