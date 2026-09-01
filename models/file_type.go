package models

type FileType string

const (
	FileTypePasswd FileType = "passwd"
	FileTypeGroup  FileType = "group"
)

// FileTypeValues is the canonical list for DB CHECK constraints.
var FileTypeValues = []string{
	string(FileTypePasswd),
	string(FileTypeGroup),
}
