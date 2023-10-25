package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"time"
)

func commitTree(treeSha string, parentCommitSha string, message string) ([20]byte, error) {
	var contents bytes.Buffer
	contents.WriteString(fmt.Sprintf("tree %s\n", treeSha))
	if parentCommitSha != "" {
		contents.WriteString(fmt.Sprintf("parent %s\n", parentCommitSha))
	}
	// get author
	timestamp := time.Now().Unix()
	timezoneOffset := time.Now().Format("-0400")
	author := fmt.Sprintf("author GT %d %s", timestamp, timezoneOffset)
	committer := fmt.Sprintf("committer GT %d %s", timestamp, timezoneOffset)
	contents.WriteString(fmt.Sprintf("author %s\n", author))
	contents.WriteString(fmt.Sprintf("committer %s\n", committer))
	contents.WriteString(fmt.Sprintf("\n%s\n", message))
	var rawSha = sha1.Sum(contents.Bytes())
	commitSha := fmt.Sprintf("%x", rawSha)
	commitPath := path.Join(".git", "objects", commitSha[:2], commitSha[2:])
	// prepend header
	header := fmt.Sprintf("commit %d\x00", contents.Len())
	var b bytes.Buffer
	b.WriteString(header)
	b.Write(contents.Bytes())
	// create zlib writer
	var compressed bytes.Buffer
	zlibWriter := zlib.NewWriter(&compressed)
	// write binary data
	if _, err := zlibWriter.Write(b.Bytes()); err != nil {
		return [20]byte{}, err
	}
	// close zlib writer
	if err := zlibWriter.Close(); err != nil {
		return [20]byte{}, err
	}
	// if file does not exist then create it, otherwise replace it
	if _, err := os.Stat(commitPath); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(".git", "objects", commitSha[:2]), 0755); err != nil {
			return [20]byte{}, err
		}
	}
	if err := os.WriteFile(commitPath, compressed.Bytes(), 0644); err != nil {
		return [20]byte{}, err
	}
	return rawSha, nil
}
