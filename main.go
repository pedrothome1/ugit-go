package main

import (
	"flag"
	"fmt"
	"os"
	"ugit-go/data"
)

func main() {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	hashObjectCmd := flag.NewFlagSet("hash-object", flag.ExitOnError)
	catFileCmd := flag.NewFlagSet("cat-file", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Println("expected subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
		doInit(initCmd.Args())
	case "hash-object":
		hashObjectCmd.Parse(os.Args[2:])
		doHashObject(hashObjectCmd.Args())
	case "cat-file":
		catFileCmd.Parse(os.Args[2:])
		doCatFile(catFileCmd.Args())
	}
}

func doInit(_ []string) {
	err := data.Initialize()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Initialized empty ugit repository in %s/%s\n", getwd(), data.UGitDir)
}

func doHashObject(args []string) {
	b, err := os.ReadFile(args[0])
	if err != nil {
		panic(err)
	}

	oid, err := data.HashObject(b, data.BlobType)
	if err != nil {
		panic(err)
	}

	fmt.Println(oid)
}

func doCatFile(args []string) {
	content, err := data.GetObject(args[0], data.NoneType)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(content))
}

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return wd
}
