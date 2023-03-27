package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
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

		_, content, err := CatFile(sha1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Printf("%s", content)

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

	case "ls-tree":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "usage: mygit ls-tree --name-only <tree_sha>\n")
			os.Exit(1)
		}

		sha := os.Args[3]

		_, content, err := CatFile(sha)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}

		names := make([]string, 0)
		for i := 0; i < len(content); {
			for content[i] != byte(' ') { // mode
				i++
			}
			i++              // space
			name := []byte{} // file/folder name
			for content[i] != byte('\u0000') {
				name = append(name, content[i])
				i++
			}
			names = append(names, string(name))
			i++     // \0
			i += 20 // SHA1 (20byte)
		}

		sort.Sort(sort.StringSlice(names))
		for _, name := range names {
			fmt.Println(name)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

// header, content, errorを返す
func CatFile(sha1 string) ([]byte, []byte, error) {
	path := filepath.Join(".git/objects", sha1[:2], sha1[2:])

	zlibContent, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("Error opening file: %s\n", err)
	}

	r, err := zlib.NewReader(zlibContent)
	if err != nil {
		return nil, nil, fmt.Errorf("Error reading file: %s\n", err)
	}
	defer r.Close()

	store, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("Error reading file: %s\n", err)
	}

	firstNullbyteIndex := strings.Index(string(store), "\u0000")
	header := store[:firstNullbyteIndex]
	content := store[firstNullbyteIndex+1:]
	return header, content, nil
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
