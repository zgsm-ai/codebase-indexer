// utils/file.go - 文件处理相关函数
package utils

import (
	"archive/zip"
	"io"
	"os"
)

// 构建 ZIP 文件
func CreateZipFile(filePaths []string, zipFileName string) error {
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, filePath := range filePaths {
		fileToZip, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		info, err := fileToZip.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 使用文件路径作为 ZIP 中的文件名
		header.Name = filePath
		if info.IsDir() {
			header.Name += "/"
		} else {
			// 正常文件的默认压缩方法
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err = io.Copy(writer, fileToZip); err != nil {
			return err
		}
	}
	return nil
}
