package internal

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type UploaderOpts struct {
	remoteRootFolderID string
	pathsToUpdate      map[string]string
	authClient         *AuthClient
	srv                *drive.Service
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

// Make sure this is invoked once
func (u *Uploader) InitUploader() error {
	// check if every localPath actually exists(validate)
	for localPath, _ := range u.opts.pathsToUpdate {
		if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("path %s does not exist in locally", localPath)
		}
	}

	// init the auth client
	auth, err := u.opts.authClient.GetAuthClient()
	if err != nil {
		return err
	}

	// init the drive service
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(auth))
	if err != nil {
		return err
	}

	u.opts.srv = srv

	return nil
}

func (u *Uploader) Upload() (err error) {
	var currentParentID string
	for localPath, remotePath := range u.opts.pathsToUpdate {
		fmt.Println("localPath = " + localPath)

		pathSegments := strings.Split(remotePath, "/")
		currentParentID = u.opts.remoteRootFolderID
		var pathSegmentsItr int

		var exists bool
		for idx, segment := range pathSegments {
			fmt.Println("segment = " + segment)
			pathSegmentsItr = idx

			exists, err = u.search(segment, &currentParentID)
			if err != nil {
				return fmt.Errorf("something happened while performing search %w", err)
			}

			if !exists {
				break
			}
		}

		f, err := os.Open(localPath)
		if err != nil {
			return fmt.Errorf("failed to open file located in %s: %w", localPath, err)
		}
		defer f.Close()

		fInfo, err := f.Stat()
		if err != nil {
			return err
		}

		if !exists {
			fmt.Printf("file/folder does not exist\n")

			err := u.createEmptyDirs(pathSegments, pathSegmentsItr, &currentParentID)
			if err != nil {
				return err
			}

			if fInfo.IsDir() {
				// check if folder exists, if not, create Folder, else, just update.
				folderExists, err := u.search(filepath.Base(f.Name()), &currentParentID)
				if err != nil {
					return err
				}

				if folderExists {

				} else {
					err := u.createFolder(f, currentParentID)
					if err != nil {
						return err
					}
				}
				continue
			}

			remoteFile := u.convertLocalToRemoteFile(f)
			remoteFile.Parents = append(remoteFile.Parents, currentParentID)

			createdFile, err := u.createFile(remoteFile, f)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully uploaded file with ID: %s\n", createdFile.Id)
			continue
		}

		if fInfo.IsDir() {
			// check if folder exists, if not, create Folder, else, just update.
			folderExists, err := u.search(filepath.Base(f.Name()), &currentParentID)
			if err != nil {
				return err
			}

			if folderExists {

			} else {
				err := u.createFolder(f, currentParentID)
				if err != nil {
					return err
				}
			}
			continue
		}

		prevParentID := currentParentID
		remoteFile := u.convertLocalToRemoteFile(f)

		updatedFile, err := u.updateFile(currentParentID, prevParentID, remoteFile, f)
		if err != nil {
			return err
		}

		fmt.Printf("Successfully uploaded file with ID: %s\n", updatedFile.Id)
	}

	return nil
}

func (u *Uploader) search(segment string, currentParentID *string) (bool, error) {
	fmt.Println("query = " + segment)
	query := fmt.Sprintf("name = '%s' and (mimeType = 'application/vnd.google-apps.folder' or mimeType != 'application/vnd.google-apps.folder') and '%s' in parents", segment, *currentParentID)

	fmt.Println(query)
	r, err := u.opts.srv.Files.List().Q(query).Do()

	if err != nil {
		return false, err
	}

	if len(r.Files) == 0 {
		return false, nil
	}

	*currentParentID = r.Files[0].Id

	fmt.Printf("Found folder '%s' with ID '%s' Count: %d Kind: %s\n", r.Files[0].Name, r.Files[0].Id, len(r.Files), r.Files[0].MimeType)

	return true, nil
}

func (u *Uploader) createEmptyDirs(pathSegments []string, startIdx int, currentParentID *string) error {
	for _, segment := range pathSegments[startIdx:] {
		emptyFolder := &drive.File{
			Name:     segment,
			MimeType: "application/vnd.google-apps.folder",
			Parents:  []string{*currentParentID},
		}

		folder, err := u.opts.srv.Files.Create(emptyFolder).Do()
		if err != nil {
			return fmt.Errorf("failed to create folder: %w", err)
		}

		*currentParentID = folder.Id
	}

	return nil
}

func (u *Uploader) createFile(remoteFile *drive.File, localFile *os.File) (*drive.File, error) {
	createdFile, err := u.opts.srv.Files.Create(remoteFile).Media(localFile).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return createdFile, nil
}

func (u *Uploader) updateFile(fileID, parentID string, remoteFile *drive.File, localFile *os.File) (*drive.File, error) {
	updatedFile, err := u.opts.srv.Files.Update(fileID, remoteFile).AddParents(parentID).Media(localFile).Do()
	if err != nil {
		return nil, err
	}
	return updatedFile, nil
}

func (u *Uploader) createFolder(folder *os.File, currentParentID string) error {
	folderIDs := make(map[string]string)

	rootFolder := &drive.File{
		Name:     filepath.Base(folder.Name()),
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{currentParentID},
	}

	createdRoot, err := u.opts.srv.Files.Create(rootFolder).Do()
	if err != nil {
		return fmt.Errorf("failed to create root folder: %w", err)
	}

	folderIDs[folder.Name()] = createdRoot.Id
	fmt.Printf("Created root directory: %s\n", folder.Name())

	root := folder.Name()
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		parentDir := filepath.Dir(path)
		parentID := folderIDs[parentDir]

		if d.IsDir() {
			emptyFolder := &drive.File{
				Name:     filepath.Base(path),
				MimeType: "application/vnd.google-apps.folder",
				Parents:  []string{parentID},
			}

			createdFolder, err := u.opts.srv.Files.Create(emptyFolder).Do()
			if err != nil {
				return fmt.Errorf("failed to create folder: %w", err)
			}

			folderIDs[path] = createdFolder.Id
			fmt.Printf("Directory: %s\n", path)
		} else {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			remoteFile := u.convertLocalToRemoteFile(file)
			remoteFile.Parents = []string{parentID}
			_, err = u.createFile(remoteFile, file)
			if err != nil {
				return err
			}

			fmt.Printf("File: %s\n", path)
		}

		return nil
	})
}

func (u *Uploader) convertLocalToRemoteFile(file *os.File) *drive.File {
	filename := filepath.Base(file.Name())
	mimeType := mime.TypeByExtension(filepath.Ext(filename))

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return &drive.File{
		Name:     filename,
		MimeType: mimeType,
	}
}
