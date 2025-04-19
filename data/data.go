package data

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type ObjectType string

const (
	NoneType   ObjectType = "none"
	BlobType   ObjectType = "blob"
	TreeType   ObjectType = "tree"
	CommitType ObjectType = "commit"
)

const UGitDir = ".ugit"

func Initialize() error {
	err := os.Mkdir(UGitDir, 0750)
	if err != nil {
		return err
	}

	err = os.Mkdir(filepath.Join(UGitDir, "objects"), 0750)
	if err != nil {
		return err
	}

	return nil
}

func SetHead(oid string) error {
	return os.WriteFile(filepath.Join(UGitDir, "HEAD"), []byte(oid), 0666)
}

func GetHead() (string, error) {
	oid, err := os.ReadFile(filepath.Join(UGitDir, "HEAD"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", err
	}

	return string(oid), nil
}

func HashObject(data []byte, typ ObjectType) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(data)+len(typ)+1))
	buf.Write([]byte(typ))
	buf.WriteByte(0)
	buf.Write(data)
	obj := buf.Bytes()

	h := sha1.New()
	h.Write(obj)
	oid := hex.EncodeToString(h.Sum(nil))

	err := os.WriteFile(filepath.Join(UGitDir, "objects", oid), obj, 0666)
	if err != nil {
		return "", err
	}

	return oid, nil
}

func GetObject(oid string, expected ObjectType) ([]byte, error) {
	obj, err := os.ReadFile(filepath.Join(UGitDir, "objects", oid))
	if err != nil {
		return nil, err
	}

	split := bytes.Split(obj, []byte{0})
	typ := ObjectType(split[0])
	content := split[1]

	if expected != NoneType {
		if expected != typ {
			return nil, fmt.Errorf("expected type %s, got %s", expected, typ)
		}
	}

	return content, nil
}
