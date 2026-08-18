package fs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/konfidence-project/konfidence/internal/kden/log"
)

type FileData struct {
	path *string
	fs.FileInfo
}

func (f *FileData) GetFilePath() string {
	if f.path == nil {
		return ""
	}
	return *f.path
}

func (f *FileData) FileExists() bool {
	return f.FileInfo != nil
}

func (f *FileData) OpenFile() (io.ReadCloser, error) {
	if f.FileExists() {
		return os.Open(*f.path)
	}
	return nil, fmt.Errorf("file %q does not exist", f.GetFilePath())
}

func (f *FileData) SetFilePath(s string) error {
	if f.path == nil {
		f.path = new(string)
	}

	extensions := []string{".yaml", ".yml"}
	found := false
	for _, ext := range extensions {
		tryPath := s
		if !strings.HasSuffix(s, ext) {
			tryPath = s + ext
		}

		info, err := os.Stat(tryPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat path %q: %w", tryPath, err)
		}
		*f.path = tryPath
		if err == nil {
			f.FileInfo = info
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("file %q not found (tried .yaml/.yml extensions)", s)
	}
	return nil
}

func ReadFile(flag *FileData) ([]byte, error) {
	path := flag.GetFilePath()
	constructorStream, err := flag.OpenFile()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}

	defer func(constructorStream io.ReadCloser) {
		err := constructorStream.Close()
		if err != nil {
			log.Errorf("failed to close file stream: %v", err)
		}
	}(constructorStream)

	constructorData, err := io.ReadAll(constructorStream)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return constructorData, nil
}

func ToFileData(paths []string) ([]*FileData, error) {
	var result []*FileData
	for _, p := range paths {
		fd := &FileData{}
		if err := fd.SetFilePath(p); err != nil {
			return nil, err
		}
		if fd.IsDir() {
			return nil, fmt.Errorf("path %q is a directory, must point to a file", p)
		}
		result = append(result, fd)
	}
	return result, nil
}
