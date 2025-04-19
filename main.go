package main

import (
	"flag"
	"fmt"
	"os"
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

	content, err := data.GetObject(args[0], data.NoneType)
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

	err := base.ReadTree(args[0])
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
		oid, err = data.GetHead()
		if err != nil {
			die(err)
		}
	} else {
		oid = logFrom
	}

	for oid != "" {
		commit, err := base.GetCommit(oid)
		if err != nil {
			die(err)
		}

		fmt.Printf("commit %s\n", oid)
		fmt.Printf("%s\n\n", commit.Message)

		oid = commit.Parent
	}
}

func doCheckout(args []string) {
	if len(args) < 1 {
		die("expected commit hash")
	}

	err := base.Checkout(args[0])
	if err != nil {
		die(err)
	}
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
