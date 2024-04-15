package modules

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

func DownloadFile(urlStr string, outputDir string) {
    Parallel([]string{urlStr}, func(url string) {
        resp, err := http.Get(url)
        if err != nil {
            fmt.Printf("Error downloading file %s: %s\n", url, err)
            return
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            fmt.Printf("Unable to download file %s: %s\n", url, resp.Status)
            return
        }

        contentDisposition := resp.Header.Get("Content-Disposition")
		regexp := regexp.MustCompile(`filename=\s*(?:"([^"]+)"|([^;]+))`)
		fileName := Filter(regexp.FindStringSubmatch(contentDisposition)[1:], func(group string) bool { return group != "" })[0]
        outFile, err := os.Create(filepath.Join(outputDir, fileName))
        if err != nil {
            fmt.Printf("Error creating file %s: %s\n", fileName, err)
            return
        }
        defer outFile.Close()

        _, err = io.Copy(outFile, resp.Body)
        if err != nil {
            fmt.Printf("Error copying content to file %s: %s\n", fileName, err)
            return
        }
    })
}

func ExtractFromZip(zipFile string, ext string, outputDir string) error {
    reader, err := zip.OpenReader(zipFile)
    if err != nil {
        return fmt.Errorf("unable to open zip file: %v", err)
    }
    defer reader.Close()

    var filesToExtract []string
    for _, file := range reader.File {
        if strings.HasSuffix(file.Name, ext) {
            filesToExtract = append(filesToExtract, file.Name)
        }
    }

    Parallel(filesToExtract, func(fileName string) {
        file, err := reader.Open(fileName)
        if err != nil {
            fmt.Printf("Error opening file %s from zip: %s\n", fileName, err)
            return
        }
        defer file.Close()

        outFile, err := os.Create(filepath.Join(outputDir, fileName))
        if err != nil {
            fmt.Printf("Error creating file %s: %s\n", fileName, err)
            return
        }
        defer outFile.Close()

        _, err = io.Copy(outFile, file)
        if err != nil {
            fmt.Printf("Error copying content to file %s: %s\n", fileName, err)
            return
        }
    })

	return nil
}

func Parallel[TYPE any](data []TYPE, f func(TYPE)) {
	var wg sync.WaitGroup
	for _, d := range data {
		wg.Add(1)
		go func(d TYPE) {
			defer wg.Done()
			f(d)
		}(d)
	}
	wg.Wait()
}

func Filter[TYPE any](data []TYPE, f func(TYPE) bool) []TYPE {
	var result []TYPE
	for _, d := range data {
		if f(d) {
			result = append(result, d)
		}
	}
	return result
}