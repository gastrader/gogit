package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	// Uncomment this block to pass the first stage!
	"os"
)

// const objectsDirectory = ".git/objects"

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

		fmt.Println("Initialized git directory.")

	case "cat-file":
		if len(os.Args) < 4 || os.Args[2] != "-p" {
			fmt.Fprintf(os.Stderr, "Usage: ./your_git.sh cat-file -p <blob_sha>\n")
			os.Exit(1)
		}
		blobSha := os.Args[3]
		objectDir := ".git/objects/" + blobSha[:2]
		objectFile := blobSha[2:]
		objectPath := filepath.Join(objectDir, objectFile)

		compressedData, err := os.ReadFile(objectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading object file: %s\n", err)
			os.Exit(1)
		}
		r, err := zlib.NewReader(bytes.NewReader(compressedData))
		if err != nil {
			panic(err)
		}
		defer r.Close()

		var output bytes.Buffer
		io.Copy(&output, r)
		outputString := strings.Split(output.String(), "\x00")[1]
		fmt.Print(outputString)

	case "hash-object":

		if len(os.Args) < 4 || os.Args[2] != "-w" {
			fmt.Fprintf(os.Stderr, "Usage: ./your_git.sh hash-object -w <filename>\n")
			os.Exit(1)
		}
		fileName := os.Args[3]
		blobSha, err := hashObject(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error hashing object: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("%x\n", blobSha)

	case "ls-tree":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: ./your_git.sh ls-tree <object>\n")
			os.Exit(1)
		}
		treeSha := os.Args[3]
		treePath := path.Join(".git", "objects", treeSha[:2], treeSha[2:])
		reader, err := os.Open(treePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
			os.Exit(1)
		}
		zlibReader, err := zlib.NewReader(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating zlib reader: %s\n", err)
			os.Exit(1)
		}
		// read binary data
		decompressedContents, err := io.ReadAll(zlibReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
			os.Exit(1)
		}
		// skip header
		decompressedContents = decompressedContents[bytes.IndexByte(decompressedContents, 0)+1:]
		var names []string
		// each entry is in the format <mode> <name>\x00<sha>, keep slicing until all the bytes are processed
		for len(decompressedContents) > 0 {
			// get mode
			mode := decompressedContents[:strings.IndexByte(string(decompressedContents), ' ')]
			decompressedContents = decompressedContents[len(mode)+1:]
			// get name
			name := decompressedContents[:strings.IndexByte(string(decompressedContents), 0)]
			decompressedContents = decompressedContents[len(name)+1:]
			// get sha
			sha := decompressedContents[:20]
			decompressedContents = decompressedContents[len(sha):]
			names = append(names, string(name))
		}
		for _, name := range names {
			fmt.Printf("%s\n", name)
		}

	case "write-tree":
		if len(os.Args) > 2 {
			fmt.Fprintf(os.Stderr, "Usage: ./your_git.sh write-tree\n")
			os.Exit(1)
		}
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %s\n", err)
			os.Exit(1)
		}
		for {
			if _, err := os.Stat(path.Join(dir, ".git")); err == nil {
				break
			}

			dir = path.Dir(dir)
		}
		treeSha, err := hashTree(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error hashing tree: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%x\n", treeSha)

	case "commit-tree":
		if len(os.Args) < 6 {
			fmt.Fprintf(os.Stderr, "Usage: ./your_git.sh commit-tree <tree_sha> -p <commit_sha> -m <message>\n")
			os.Exit(1)
		}
		treeSha := os.Args[2]
		parentCommitSha := os.Args[4]
		message := os.Args[6]
		commitSha, err := commitTree(treeSha, parentCommitSha, message)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error committing tree: %s\n", err)
			os.Exit(1)
		}
		// print sha
		fmt.Printf("%x\n", commitSha)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
