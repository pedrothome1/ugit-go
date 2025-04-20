package data

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
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

const (
	HeadRef string = "HEAD"
)

const UGitDir = ".ugit"

type RefData struct {
	Name string
	OID  string
}

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

func UpdateRef(ref, oid string) error {
	refPath := filepath.Join(UGitDir, ref)

	err := os.MkdirAll(filepath.Dir(refPath), 0750)
	if err != nil {
		return err
	}

	return os.WriteFile(refPath, []byte(oid), 0666)
}

func GetRef(ref string) (string, error) {
	oid, err := os.ReadFile(filepath.Join(UGitDir, ref))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", err
	}

	return string(oid), nil
}

func AllRefs() ([]*RefData, error) {
	var result []*RefData

	refs := []string{HeadRef}

	err := filepath.Walk(filepath.Join(UGitDir, "refs"), func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			relPath, err := filepath.Rel(UGitDir, path)
			if err != nil {
				return err
			}

			refs = append(refs, relPath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, ref := range refs {
		oid, err := GetRef(ref)
		if err != nil {
			return nil, err
		}

		result = append(result, &RefData{
			Name: ref,
			OID:  oid,
		})
	}

	return result, nil
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
