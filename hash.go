package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
)

func hashObject(filePath string) ([20]byte, error) {
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return [20]byte{}, err
	}
	// prepend header
	header := fmt.Sprintf("blob %d\x00", len(fileContents))
	storeContents := append([]byte(header), fileContents...)
	rawSha := sha1.Sum(storeContents)
	blobSha := fmt.Sprintf("%x", rawSha)
	blobPath := path.Join(".git", "objects", blobSha[:2], blobSha[2:])
	// create zlib writer
	var b bytes.Buffer
	zlibWriter := zlib.NewWriter(&b)
	// write binary data
	if _, err := zlibWriter.Write(storeContents); err != nil {
		return [20]byte{}, err
	}
	// close zlib writer
	if err := zlibWriter.Close(); err != nil {
		return [20]byte{}, err
	}
	// if file does not exist then create it, otherwise replace it
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(".git", "objects", blobSha[:2]), 0755); err != nil {
			return [20]byte{}, err
		}
	}
	if err := os.WriteFile(blobPath, b.Bytes(), 0644); err != nil {
		return [20]byte{}, err
	}
	return rawSha, nil
}
func hashTree(rootPath string) ([20]byte, error) {
	files, err := os.ReadDir(rootPath)
	if err != nil {
		return [20]byte{}, err
	}
	var entries []string
	for _, file := range files {
		// skip .git directory
		if file.Name() == ".git" {
			continue
		}
		var sha [20]byte
		mode := 0o100644
		fullFilePath := path.Join(rootPath, file.Name())
		if file.IsDir() {
			treeSha, err := hashTree(fullFilePath)
			if err != nil {
				return [20]byte{}, err
			}
			sha = treeSha
			// octal representation of directory (octal type)
			mode = 0o040000
		} else {
			// get file sha
			fileSha, err := hashObject(fullFilePath)
			if err != nil {
				return [20]byte{}, err
			}
			sha = fileSha
			// octal representation of file (regular type)
			mode = 0o100644
		}
		entries = append(entries, fmt.Sprintf("%o %s\x00%s", mode, file.Name(), sha))
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i][strings.IndexByte(entries[i], ' ')+1:] < entries[j][strings.IndexByte(entries[j], ' ')+1:]
	})
	// create tree object
	var b bytes.Buffer
	var contents bytes.Buffer
	for _, entry := range entries {
		contents.WriteString(entry)
	}
	header := fmt.Sprintf("tree %d\x00", contents.Len())
	b.WriteString(header)
	b.Write(contents.Bytes())
	var rawSha = sha1.Sum(b.Bytes())
	treeSha := fmt.Sprintf("%x", rawSha)
	treePath := path.Join(".git", "objects", treeSha[:2], treeSha[2:])
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
	if _, err := os.Stat(treePath); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(".git", "objects", treeSha[:2]), 0755); err != nil {
			return [20]byte{}, err
		}
	}
	if err := os.WriteFile(treePath, compressed.Bytes(), 0644); err != nil {
		return [20]byte{}, err
	}
	return rawSha, nil
}
