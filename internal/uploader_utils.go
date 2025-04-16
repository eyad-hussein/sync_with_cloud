package internal

type FolderOrFile int

const (
	Folder FolderOrFile = iota
	File
)

type Pair struct {
	fileOrFolder FolderOrFile
	isUploaded   bool
}

func extractFilesAndFolders(paths []string) map[string]Pair {
	return nil
}
