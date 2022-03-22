package token

import (
	"encoding/gob"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// Storage is interface for accessing token data.
type Storage interface {
	Read() (oauth2.Token, error)
	Write(oauth2.Token) error
}

type fileStorage struct {
	filename string
}

// NewFileStorage returns token storage in file.
func NewFileStorage(filename string) Storage {
	return &fileStorage{
		filename: filename,
	}
}

func (fs *fileStorage) Read() (oauth2.Token, error) {
	var out oauth2.Token
	f, err := os.Open(fs.filename)
	if err != nil {
		return out, errors.Wrapf(err, "unable to open file for reading: %s", fs.filename)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	err = dec.Decode(&out)
	if err != nil {
		return out, errors.Wrap(err, "error decoding auth file")
	}

	return out, nil
}

// Write truncates the file and replaces it with supplied token.
func (fs *fileStorage) Write(token oauth2.Token) error {
	f, err := os.Create(fs.filename)
	if err != nil {
		return errors.Wrapf(err, "unable to create file: %s", fs.filename)
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return errors.Wrap(enc.Encode(token), "error encoding auth file")
}
