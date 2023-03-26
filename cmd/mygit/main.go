package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Usage: your_git.sh <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/master\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")

	case "cat-file":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <blob_sha>\n")
			os.Exit(1)
		}

		sha1 := os.Args[3]
		path := filepath.Join(".git/objects", sha1[:2], sha1[2:])

		zlibContent, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
			os.Exit(1)
		}

		r, err := zlib.NewReader(zlibContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
			os.Exit(1)
		}
		defer r.Close()

		store, err := ioutil.ReadAll(r)
		content := strings.Split(string(store), "\u0000")[1]
		fmt.Print(content)

	case "hash-object":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <file>\n")
			os.Exit(1)
		}

		file := os.Args[3]
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
			os.Exit(1)
		}

		header := fmt.Sprintf("blob %d\u0000", len(content))
		store := header + string(content)

		sha1 := fmt.Sprintf("%x", SHA1Digest(store))
		zlibContent := ZlibDeflate(store)

		path := filepath.Join(".git/objects", sha1[:2], sha1[2:])
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(path, zlibContent, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
			os.Exit(1)
		}

		fmt.Println(sha1)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

func SHA1Digest(s string) []byte {
	h := sha1.New()
	h.Write([]byte(s))
	return h.Sum(nil)
}

func ZlibDeflate(s string) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write([]byte(s))
	w.Close()
	return b.Bytes()
}
