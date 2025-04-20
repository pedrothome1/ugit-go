package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"ugit-go/base"
	"ugit-go/data"
)

func main() {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	hashObjectCmd := flag.NewFlagSet("hash-object", flag.ExitOnError)

	catFileCmd := flag.NewFlagSet("cat-file", flag.ExitOnError)

	writeTreeCmd := flag.NewFlagSet("write-tree", flag.ExitOnError)

	readTreeCmd := flag.NewFlagSet("read-tree", flag.ExitOnError)

	commitCmd := flag.NewFlagSet("commit", flag.ExitOnError)
	var commitMsg string
	commitCmd.StringVar(&commitMsg, "m", "", "commit message")
	commitCmd.StringVar(&commitMsg, "message", "", "commit message")

	logCmd := flag.NewFlagSet("log", flag.ExitOnError)
	var logFrom string
	logCmd.StringVar(&logFrom, "oid", "", "oid to start logging history (default is HEAD)")

	checkoutCmd := flag.NewFlagSet("checkout", flag.ExitOnError)

	tagCmd := flag.NewFlagSet("tag", flag.ExitOnError)

	kCmd := flag.NewFlagSet("k", flag.ExitOnError)

	if len(os.Args) < 2 {
		die("expected subcommand")
	}

	subArgs := os.Args[2:]

	switch os.Args[1] {
	case initCmd.Name():
		initCmd.Parse(subArgs)
		doInit(initCmd.Args())
	case hashObjectCmd.Name():
		hashObjectCmd.Parse(subArgs)
		doHashObject(hashObjectCmd.Args())
	case catFileCmd.Name():
		catFileCmd.Parse(subArgs)
		doCatFile(catFileCmd.Args())
	case writeTreeCmd.Name():
		writeTreeCmd.Parse(subArgs)
		doWriteTree(writeTreeCmd.Args())
	case readTreeCmd.Name():
		readTreeCmd.Parse(subArgs)
		doReadTree(readTreeCmd.Args())
	case commitCmd.Name():
		commitCmd.Parse(subArgs)
		doCommit(commitMsg)
	case logCmd.Name():
		logCmd.Parse(subArgs)
		doLog(logFrom)
	case checkoutCmd.Name():
		checkoutCmd.Parse(subArgs)
		doCheckout(checkoutCmd.Args())
	case tagCmd.Name():
		tagCmd.Parse(subArgs)
		doTag(tagCmd.Args())
	case kCmd.Name():
		kCmd.Parse(subArgs)
		doK(kCmd.Args())
	}
}

func doInit(_ []string) {
	err := data.Initialize()
	if err != nil {
		die(err)
	}

	fmt.Printf("Initialized empty ugit repository in %s/%s\n", getwd(), data.UGitDir)
}

func doHashObject(args []string) {
	if len(args) < 1 {
		die("expected file path")
	}

	b, err := os.ReadFile(args[0])
	if err != nil {
		die(err)
	}

	oid, err := data.HashObject(b, data.BlobType)
	if err != nil {
		die(err)
	}

	fmt.Println(oid)
}

func doCatFile(args []string) {
	if len(args) < 1 {
		die("expected object hash")
	}

	content, err := data.GetObject(mustGetOID(args[0]), data.NoneType)
	if err != nil {
		die(err)
	}

	fmt.Println(string(content))
}

func doWriteTree(args []string) {
	if len(args) < 1 {
		die("expected directory path")
	}

	oid, err := base.WriteTree(args[0])
	if err != nil {
		die(err)
	}

	fmt.Println(oid)
}

func doReadTree(args []string) {
	if len(args) < 1 {
		die("expected tree hash")
	}

	err := base.ReadTree(mustGetOID(args[0]))
	if err != nil {
		die(err)
	}
}

func doCommit(msg string) {
	if msg == "" {
		fmt.Println("the commit message is required")
		return
	}

	commitOID, err := base.Commit(msg)
	if err != nil {
		die(err)
	}

	fmt.Println(commitOID)
}

func doLog(logFrom string) {
	var oid string
	var err error

	if logFrom == "" {
		oid, err = data.GetRef(data.HeadRef)
		if err != nil {
			die(err)
		}
	} else {
		oid = mustGetOID(logFrom)
	}

	all, err := base.AllCommitsAndParents([]string{oid})
	if err != nil {
		die(err)
	}

	for _, oid := range all {
		commit, err := base.GetCommit(oid)
		if err != nil {
			die(err)
		}

		fmt.Printf("commit %s\n", oid)
		fmt.Printf("%s\n\n", commit.Message)
	}
}

func doCheckout(args []string) {
	if len(args) < 1 {
		die("expected commit hash")
	}

	err := base.Checkout(mustGetOID(args[0]))
	if err != nil {
		die(err)
	}
}

func doTag(args []string) {
	if len(args) < 1 {
		die("tag name is required")
	}

	tag := args[0]
	var oid string

	if len(args) > 1 {
		oid = mustGetOID(args[1])
	}

	if oid == "" {
		head, err := data.GetRef(data.HeadRef)
		if err != nil {
			die(err)
		}

		oid = head
	}

	err := base.CreateTag(tag, oid)
	if err != nil {
		die(err)
	}
}

func doK(_ []string) {
	var dot strings.Builder
	dot.WriteString("digraph commits {\n")

	refs, err := data.AllRefs()
	if err != nil {
		die(err)
	}

	oidsSet := make(map[string]struct{})

	for _, ref := range refs {
		dot.WriteString(fmt.Sprintf("\"%s\" [shape=note]\n", ref.Name))
		dot.WriteString(fmt.Sprintf("\"%s\" -> \"%s\"\n", ref.Name, ref.OID))
		oidsSet[ref.OID] = struct{}{}
	}

	all, err := base.AllCommitsAndParents(slices.Collect(maps.Keys(oidsSet)))
	if err != nil {
		die(err)
	}

	for _, oid := range all {
		commit, err := base.GetCommit(oid)
		if err != nil {
			die(err)
		}

		dot.WriteString(fmt.Sprintf("\"%s\" [shape=box style=filled label=\"%s\"]\n", oid, oid[:10]))

		if commit.Parent != "" {
			dot.WriteString(fmt.Sprintf("\"%s\" -> \"%s\"\n", oid, commit.Parent))
		}
	}

	dot.WriteString("}")
	fmt.Println(dot.String())
}

func mustGetOID(name string) string {
	oid, err := base.GetOID(name)
	if err != nil {
		die(err)
	}

	if oid == "" {
		die(fmt.Errorf("%q ref not found", name))
	}

	return oid
}

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return wd
}

func die(msg any) {
	fmt.Println(msg)
	os.Exit(1)
}
