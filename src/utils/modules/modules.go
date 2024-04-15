package modules

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(path string) (string, error) {
	file := filepath.Join(path)
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	return string(data), nil
}

func WriteFile(path string, data string) error {
	file := filepath.Join(path)
	err := os.WriteFile(file, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func DownloadFile(urlStr string, outputDir string) error {
    resp, err := http.Get(urlStr)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unable to download file: %s", resp.Status)
    }
    
	contentDisposition := resp.Header.Get("Content-Disposition")
	fileName := strings.Trim(contentDisposition[len("attachment; filename="):], "\"")
    outFile, err := os.Create(filepath.Join(outputDir, fileName))
    if err != nil {
        return err
    }
    defer outFile.Close()
    
    _, err = io.Copy(outFile, resp.Body)
    if err != nil {
        return err
    }
    
    return nil
}

func ExtractFromZip(zipFile string, ext string, outputDir string) error {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("unable to open zip file: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ext) {
			src, err := file.Open()
			if err != nil {
				return err
			}
			defer src.Close()

			dstPath := filepath.Join(outputDir, file.Name)
			dst, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, src)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func MakeDir(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	return nil
}

func RemoveDir(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("error removing directory: %v", err)
	}

	return nil
}