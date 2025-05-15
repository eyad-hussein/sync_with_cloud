package internal

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type UploaderOpts struct {
	cfg        Config
	authClient *AuthClient
	srv        *drive.Service
}

func NewUploaderOpts(cfg Config, authClient AuthClient) UploaderOpts {
	return UploaderOpts{
		cfg:        cfg,
		authClient: &authClient,
	}
}

type Uploader struct {
	opts *UploaderOpts
}

func NewUploader(opts UploaderOpts) *Uploader {
	return &Uploader{opts: &opts}
}

// TODO:: Make sure this is invoked once
func (u *Uploader) InitUploader() error {

	if err := u.opts.cfg.ValidateConfig(); err != nil {
		return err
	}

	auth, err := u.opts.authClient.GetAuthClient()
	if err != nil {
		return err
	}

	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(auth))
	if err != nil {
		return err
	}

	u.opts.srv = srv

	return nil
}

func (u *Uploader) Upload() (err error) {
	var currentParentID string
	for localPath, remotePath := range u.opts.cfg.Paths {
		slog.Info("scanning localPath", "path", localPath)

		if _, exists := u.opts.cfg.Exclude[localPath]; exists {
			slog.Info("found localPath in exclude section, skipping", "localPath", localPath)
			continue
		}

		pathSegments := strings.Split(remotePath, "/")
		currentParentID = u.opts.cfg.RootFolderId
		var pathSegmentsItr int
		var exists bool

		for idx, segment := range pathSegments {
			pathSegmentsItr = idx

			var fileID string
			exists, fileID, err = u.search(segment, currentParentID)
			if err != nil {
				return fmt.Errorf("something happened while performing search %w", err)
			}

			if !exists {
				break
			}
			currentParentID = fileID
		}

		f, err := os.Open(localPath)
		if err != nil {
			return fmt.Errorf("failed to open file located in %s: %w", localPath, err)
		}
		defer func() {
			if closeErr := f.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}()

		fInfo, err := f.Stat()
		if err != nil {
			return err
		}

		if !exists {
			slog.Info("file/folder does not exist remotely")

			parentID := currentParentID
			err := u.createEmptyDirs(pathSegments, pathSegmentsItr, &parentID)
			if err != nil {
				return err
			}
			currentParentID = parentID

			if fInfo.IsDir() {

				err := u.createFolder(f, currentParentID)
				if err != nil {
					return err
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
			exists, newCurrentParentID, err := u.search(filepath.Base(f.Name()), currentParentID)
			if err != nil {
				return err
			}

			if !exists {
				slog.Info("currentParentId", "parentId", currentParentID)

				err := u.createFolder(f, currentParentID)
				if err != nil {
					return err
				}

				fmt.Println("Folder created successfully")
				continue
			}

			err = u.updateFolder(f, newCurrentParentID)
			if err != nil {
				return err
			}
			continue
		}

		_, remoteFileId, err := u.search(filepath.Base(f.Name()), currentParentID)
		if err != nil {
			return err
		}

		remoteFile := u.convertLocalToRemoteFile(f)
		updatedFile, err := u.updateFile(remoteFileId, currentParentID, remoteFile, f)
		if err != nil {
			return err
		}

		fmt.Printf("Successfully uploaded file with ID: %s\n", updatedFile.Id)
	}

	return nil
}

func (u *Uploader) search(query string, currentParentID string) (bool, string, error) {
	slog.Info("searching in gdrive", "query", query)
	formattedQuery := fmt.Sprintf("name = '%s' and (mimeType = 'application/vnd.google-apps.folder' or mimeType != 'application/vnd.google-apps.folder') and '%s' in parents", query, currentParentID)

	r, err := u.opts.srv.Files.List().Q(formattedQuery).Do()
	if err != nil {
		return false, "", fmt.Errorf("failed to search: %w", err)
	}

	if len(r.Files) == 0 {
		return false, "", nil
	}

	slog.Info("folder/file is found", "name", r.Files[0].Name, "ID", r.Files[0].Id)
	return true, r.Files[0].Id, nil
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

	root := folder.Name()
	folderIDs[root] = createdRoot.Id

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		slog.Info("msg", "path", path)
		if _, excluded := u.opts.cfg.Exclude[path]; excluded {
			slog.Info("skipping excluded path during folder creation", "path", path)
			if d.IsDir() {
				return filepath.SkipDir
			}
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
		} else {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				if closeErr := file.Close(); closeErr != nil && err == nil {
					err = closeErr
				}
			}()

			remoteFile := u.convertLocalToRemoteFile(file)
			remoteFile.Parents = []string{parentID}
			_, err = u.createFile(remoteFile, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (u *Uploader) buildRemoteFolderStructure(remoteFileMap map[[2]string]*drive.File, parentId string) error {
	if remoteFileMap == nil {
		remoteFileMap = make(map[[2]string]*drive.File)
	}

	query := fmt.Sprintf("'%s' in parents", parentId)

	r, err := u.opts.srv.Files.List().
		Q(query).
		Fields("files(id, name, mimeType, size, modifiedTime)").
		Do()

	if err != nil {
		return err
	}

	for _, file := range r.Files {
		key := [2]string{parentId, file.Name}
		remoteFileMap[key] = file

		slog.Info("added to map", "remoteFile", file.Name, "mimeType", file.MimeType)

		if file.MimeType == "application/vnd.google-apps.folder" {
			if err := u.buildRemoteFolderStructure(remoteFileMap, file.Id); err != nil {
				return err
			}
		}
	}

	return nil
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

func (u *Uploader) updateFolder(localFolder *os.File, currentParentID string) error {
	remoteFileMap := make(map[[2]string]*drive.File)
	err := u.buildRemoteFolderStructure(remoteFileMap, currentParentID)
	if err != nil {
		return err
	}

	processedFiles := make(map[[2]string]bool)

	root := localFolder.Name()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		if _, excluded := u.opts.cfg.Exclude[path]; excluded {
			slog.Info("skipping excluded path during folder update", "path", path)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		parentPath := filepath.Dir(path)
		var parentID string

		if parentPath == "." || parentPath == root {
			parentID = currentParentID
		} else {
			exists, id, err := u.search(filepath.Base(parentPath), currentParentID)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("parent folder not found: %s", parentPath)
			}
			parentID = id
		}

		key := [2]string{parentID, filepath.Base(path)}
		remoteFile, exists := remoteFileMap[key]

		if d.IsDir() {
			if !exists {
				slog.Info("creating directory", "path", relPath)
				folder := &drive.File{
					Name:     filepath.Base(path),
					MimeType: "application/vnd.google-apps.folder",
					Parents:  []string{parentID},
				}

				created, err := u.opts.srv.Files.Create(folder).Do()
				if err != nil {
					return fmt.Errorf("failed to create folder %s: %w", relPath, err)
				}

				remoteFileMap[key] = created
			}
		} else {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				if closeErr := file.Close(); closeErr != nil && err == nil {
					err = closeErr
				}
			}()

			if !exists {
				slog.Info("creating file", "path", relPath)

				remoteFile := u.convertLocalToRemoteFile(file)
				remoteFile.Parents = []string{parentID}

				_, err = u.createFile(remoteFile, file)
				if err != nil {
					return err
				}
			} else {
				slog.Info("updating file", "path", relPath, "id", remoteFile.Id)

				_, err := file.Stat()
				if err != nil {
					return err
				}

				newFile := u.convertLocalToRemoteFile(file)
				_, err = u.updateFile(remoteFile.Id, parentID, newFile, file)
				if err != nil {
					return err
				}
			}
		}

		processedFiles[key] = true
		return nil
	})

	if err != nil {
		return err
	}

	for key, remoteFile := range remoteFileMap {
		if !processedFiles[key] {
			slog.Info("deleting remote file", "name", remoteFile.Name)
			err := u.opts.srv.Files.Delete(remoteFile.Id).Do()
			if err != nil {
				slog.Error("failed to delete file", "name", remoteFile.Name, "error", err)
			}
		}
	}

	return nil
}
