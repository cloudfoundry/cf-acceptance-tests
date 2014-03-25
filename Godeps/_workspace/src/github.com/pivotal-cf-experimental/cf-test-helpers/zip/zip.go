package zip

import (
  "archive/zip"
  "io"
  "os"
  "path/filepath"
)

func Zip(dirOrZipFile string, targetFile *os.File) (err error) {
	if _, err := zip.OpenReader(dirOrZipFile); err == nil {
		err = CopyPathToWriter(dirOrZipFile, targetFile)
	} else {
		err = writeZipFile(dirOrZipFile, targetFile)
	}
	targetFile.Seek(0, os.SEEK_SET)
	return
}

func writeZipFile(dir string, targetFile *os.File) error {
	writer := zip.NewWriter(targetFile)
	defer writer.Close()

	return filepath.Walk(dir, func(fileName string, fileInfo os.FileInfo, err error) error {
		header, err := zip.FileInfoHeader(fileInfo)
		header.Name = filepath.ToSlash(fileName)
		if err != nil {
			return err
		}

		zipFilePart, err := writer.CreateHeader(header)
		err = CopyPathToWriter(fileName, zipFilePart)
		return nil
	})
}

func CopyPathToWriter(originalFilePath string, targetWriter io.Writer) (err error) {
	originalFile, err := os.Open(originalFilePath)
	if err != nil {
		return
	}
	defer originalFile.Close()

	_, err = io.Copy(targetWriter, originalFile)
	if err != nil {
		return
	}

	return
}
