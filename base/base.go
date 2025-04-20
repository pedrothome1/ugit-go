package base

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/gammazero/deque"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"ugit-go/data"
	"unicode"
)

type CommitData struct {
	Tree    string
	Parent  string
	Message string
}

type treeEntry struct {
	Type data.ObjectType
	OID  string
	Name string
}

func (t *treeEntry) String() string {
	return fmt.Sprintf("%s %s %s", t.Type, t.OID, t.Name)
}

func WriteTree(dir string) (string, error) {
	var entries []treeEntry

	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		full := filepath.Join(dir, file.Name())

		if isIgnored(full) {
			continue
		}

		var oid string
		var typ data.ObjectType

		if file.IsDir() {
			typ = data.TreeType

			oid, err = WriteTree(full)
			if err != nil {
				return "", err
			}
		} else {
			typ = data.BlobType

			b, err := os.ReadFile(full)
			if err != nil {
				return "", err
			}

			oid, err = data.HashObject(b, data.BlobType)
			if err != nil {
				return "", err
			}
		}

		entries = append(entries, treeEntry{
			Type: typ,
			OID:  oid,
			Name: file.Name(),
		})
	}

	slices.SortFunc(entries, func(a, b treeEntry) int {
		return strings.Compare(a.String(), b.String())
	})

	buf := new(bytes.Buffer)
	for _, entry := range entries {
		buf.WriteString(fmt.Sprintf("%s\n", entry.String()))
	}

	treeOID, err := data.HashObject(buf.Bytes(), data.TreeType)
	if err != nil {
		return "", err
	}

	return treeOID, nil
}

func ReadTree(treeOID string) error {
	err := emptyCurrentDir()
	if err != nil {
		return err
	}

	tree, err := getTree(treeOID, ".")
	if err != nil {
		return err
	}

	for path, oid := range tree {
		err = os.MkdirAll(filepath.Dir(path), 0750)
		if err != nil {
			return err
		}

		obj, err := data.GetObject(oid, data.BlobType)
		if err != nil {
			return nil
		}

		err = os.WriteFile(path, obj, 0666)
		if err != nil {
			return nil
		}
	}

	return nil
}

func Commit(msg string) (string, error) {
	treeOID, err := WriteTree(".")
	if err != nil {
		return "", err
	}

	head, err := data.GetRef(data.HeadRef)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("tree %s\n", treeOID))

	if head != "" {
		buf.WriteString(fmt.Sprintf("parent %s\n", head))
	}

	buf.WriteRune('\n')
	buf.WriteString(msg + "\n")

	commitOID, err := data.HashObject(buf.Bytes(), data.CommitType)
	if err != nil {
		return "", err
	}

	err = data.UpdateRef(data.HeadRef, commitOID)
	if err != nil {
		return "", err
	}

	return commitOID, nil
}

func GetCommit(oid string) (*CommitData, error) {
	var parent string
	var tree string
	var messageLines []string

	commit, err := data.GetObject(oid, data.CommitType)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(commit), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if line == "" {
			for j := i + 1; j < len(lines); j++ {
				messageLines = append(messageLines, lines[j])
			}
			break
		}

		split := strings.SplitN(line, " ", 2)
		key := split[0]
		val := split[1]

		if key == "tree" {
			tree = val
		} else if key == "parent" {
			parent = val
		} else {
			return nil, fmt.Errorf("unknown key %q in commit %q", key, oid)
		}
	}

	message := strings.Join(messageLines, "\n")

	return &CommitData{
		Tree:    tree,
		Parent:  parent,
		Message: message,
	}, nil
}

func Checkout(oid string) error {
	commit, err := GetCommit(oid)
	if err != nil {
		return err
	}

	err = ReadTree(commit.Tree)
	if err != nil {
		return err
	}

	err = data.UpdateRef(data.HeadRef, oid)
	if err != nil {
		return err
	}

	return nil
}

func CreateTag(name, oid string) error {
	return data.UpdateRef(filepath.Join("refs", "tags", name), oid)
}

func AllCommitsAndParents(oids []string) ([]string, error) {
	var oidsDeque deque.Deque[string]
	for _, oid := range oids {
		oidsDeque.PushBack(oid)
	}

	visitedSet := make(map[string]struct{}, len(oids))

	var result []string

	for oidsDeque.Len() > 0 {
		oid := oidsDeque.PopFront()

		if _, ok := visitedSet[oid]; oid == "" || ok {
			continue
		}

		visitedSet[oid] = struct{}{}
		result = append(result, oid)

		commit, err := GetCommit(oid)
		if err != nil {
			return nil, err
		}

		oidsDeque.PushFront(commit.Parent)
	}

	return result, nil
}

func GetOID(name string) (string, error) {
	if name == "@" {
		name = data.HeadRef
	}

	tryRefs := []string{
		name,
		filepath.Join("refs", name),
		filepath.Join("refs", "tags", name),
		filepath.Join("refs", "heads", name),
	}

	for _, ref := range tryRefs {
		oid, err := data.GetRef(ref)
		if err != nil {
			return "", err
		}

		if oid != "" {
			return oid, nil
		}
	}

	if len(name) != sha1.Size*2 {
		return "", fmt.Errorf("%q size is not a sha1 hex digest size", name)
	}

	for _, r := range name {
		if !strings.ContainsRune("0123456789abcdef", unicode.ToLower(r)) {
			return "", fmt.Errorf("%q is not a valid sha1 hex digest", name)
		}
	}

	return name, nil
}

func emptyCurrentDir() error {
	err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if path == "." || isIgnored(path) {
			return nil
		}

		return os.RemoveAll(path)
	})
	if err != nil {
		return err
	}

	return nil
}

func getTree(treeOID, basePath string) (map[string]string, error) {
	result := make(map[string]string)

	entries, err := treeEntries(treeOID)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name, "/") {
			panic("unexpected '/' in entry name")
		}
		if slices.Contains([]string{".", ".."}, entry.Name) {
			panic(fmt.Sprintf("unexpected %q as entry name", entry.Name))
		}

		path := filepath.Join(basePath, entry.Name)

		if entry.Type == data.BlobType {
			result[path] = entry.OID
		} else if entry.Type == data.TreeType {
			t, err := getTree(entry.OID, path)
			if err != nil {
				return nil, err
			}

			maps.Insert(result, maps.All(t))
		} else {
			return nil, fmt.Errorf("unexpected tree entry type %q", entry.Type)
		}
	}

	return result, nil
}

func treeEntries(oid string) ([]treeEntry, error) {
	obj, err := data.GetObject(oid, data.TreeType)
	if err != nil {
		return nil, err
	}

	var result []treeEntry

	lines := strings.Split(string(obj), "\n")
	for _, line := range lines {
		if line != "" {
			entry := strings.Split(line, " ")

			result = append(result, treeEntry{
				Type: data.ObjectType(entry[0]),
				OID:  entry[1],
				Name: entry[2],
			})
		}
	}

	return result, nil
}

func isIgnored(path string) bool {
	splitPath := strings.Split(path, string([]rune{filepath.Separator}))

	return slices.Contains(splitPath, data.UGitDir)
}
