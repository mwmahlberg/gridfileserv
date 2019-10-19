package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileStore represents a filesystem based file Repository.
type FileStore struct {
	basepath string
}

// NewFile creates a new filesystem based file Repository.
func NewFile(base string) (*FileStore, error) {
	f, err := os.Stat(base)

	// If the directory does not exist...
	if os.IsNotExist(err) {
		//  ...we try to create it.
		if err := os.Mkdir(base, 0750); err != nil {
			return nil, fmt.Errorf("Error creating '%s': %s", base, err)
		}
	} else if err != nil {
		// Could be a permission problem, let's check.
		return nil, handleErr(base, err)
	} else if !f.IsDir() {
		// It exists and we can access it, but it is simply not a directory.
		return nil, fmt.Errorf("'%s' is not a directory", base)
	}

	// All good.
	store := &FileStore{basepath: base}

	return store, nil
}

// StreamFrom statisfies the Retriever interface.
func (s *FileStore) StreamFrom(path string, w io.Writer) (err error) {

	f, err := os.OpenFile(filepath.Join(s.basepath, path), os.O_RDONLY|os.O_EXCL, 0640)
	if err != nil {
		return handleErr(path, err)
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// StreamTo satisfies the Storer interface.
func (s *FileStore) StreamTo(path string, source io.Reader) (err error) {

	var f *os.File
	f, err = os.OpenFile(filepath.Join(s.basepath, path), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModeExclusive|0640)
	if err != nil {
		return handleErr(path, err)
	}
	defer f.Close()
	_, err = io.Copy(f, source)
	return nil
}

func handleErr(path string, err error) error {
	switch err {
	case os.ErrNotExist:
		return err
	case os.ErrPermission:
		return fmt.Errorf("Wrong permissions to access '%s'", path)
	default:
		return fmt.Errorf("accessing '%s': %s", path, err)
	}
}

// Close satisfies the Repository interface, but is a noop for a filesystem based repo.
func (s *FileStore) Close() error {
	return nil
}
