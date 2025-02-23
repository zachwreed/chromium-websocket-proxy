package chromeprofile

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unzipSource(source, destination string) (profile string, err error) {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return profile, err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return profile, err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for idx, file := range reader.File {
		if idx == 0 {
			profile = file.Name
		}
		err = unzipFile(file, destination)
		if err != nil {
			return profile, err
		}
	}
	profile, _ = strings.CutSuffix(profile, "/")
	return profile, nil
}

func unzipFile(file *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, file.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := file.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err = io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}
