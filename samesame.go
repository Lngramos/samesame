package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type SHA256Checksum [32]byte

type DuplicateFinder struct {
	filesByChecksum map[SHA256Checksum][]string
	duplicates      map[SHA256Checksum]struct{}
}

func newDuplicateFinder() DuplicateFinder {
	return DuplicateFinder{
		filesByChecksum: make(map[SHA256Checksum][]string),
		duplicates:      make(map[SHA256Checksum]struct{}),
	}
}

func readFileForSHA256(path string) (SHA256Checksum, error) {
	// TODO: stream file instead of loading it into memory!
	data, err := os.ReadFile(path)
	if err != nil {
		return SHA256Checksum{}, err
	}
	return sha256.Sum256(data), nil
}

func (a *DuplicateFinder) ComputeFileChecksums(filePath string, d fs.DirEntry, err error) error {
	if d.Type().IsRegular() {
		checksum, err := readFileForSHA256(filePath)
		if err != nil {
			// skip file
			log.Printf("[WARN] %s", err.Error())
			return nil
		}

		if files, ok := a.filesByChecksum[checksum]; ok {
			if _, alreadyMarkedAsDuplicate := a.duplicates[checksum]; !alreadyMarkedAsDuplicate {
				// mark as duplicate
				a.duplicates[checksum] = struct{}{}
			}

			a.filesByChecksum[checksum] = append(files, filePath)
		} else {
			a.filesByChecksum[checksum] = []string{filePath}
		}
	}

	return nil
}

func main() {
	rootDir := flag.String("path", ".", "directory to search for duplicate files in")
	flag.Parse()

	df := newDuplicateFinder()

	rootDirPath := path.Clean(*rootDir)

	fmt.Printf("Scanning for duplicate files in dir: %s\n\n", rootDirPath)

	if err := filepath.WalkDir(rootDirPath, df.ComputeFileChecksums); err != nil {
		panic(err)
	}

	for checksum := range df.duplicates {
		fmt.Printf("Found duplicate files (%x): %s\n", checksum, strings.Join(df.filesByChecksum[checksum], ", "))
	}
}
