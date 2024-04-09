package downloader

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func DownloadAndExtract(downloadUrl, outputDir string, ext string) error {
	url, err := url.Parse(downloadUrl)
	if err != nil {
		return fmt.Errorf("error parsing URL: %v", err)
	}

	tempZipFile := filepath.Join(outputDir, "temp.zip")
	err = downloadFile(*url, tempZipFile)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}

	err = extractFiles(tempZipFile, outputDir, ext)
	if err != nil {
		return fmt.Errorf("error extracting files: %v", err)
	}

	os.Remove(tempZipFile)

	return nil
}

func downloadFile(url url.URL, filePath string) error {
	resp, err := http.Get(url.String())
	if err != nil {
		return fmt.Errorf("error downloading: %v", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	return nil
}

func extractFiles(zipFile, outputDir, ext string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("error opening zip file: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ext) {
			err := getFile(f, outputDir)
			if err != nil {
				return fmt.Errorf("error extracting file: %v", err)
			}
		}
	}

	return nil
}

func getFile(file *zip.File, outputDir string) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("error opening file in zip: %v", err)
	}
	defer rc.Close()

	extracted := filepath.Join(outputDir, file.Name)
	extractedFile, err := os.Create(extracted)
	if err != nil {
		return fmt.Errorf("error creating extracted file: %v", err)
	}
	defer extractedFile.Close()

	_, err = io.Copy(extractedFile, rc)
	if err != nil {
		return fmt.Errorf("error writing to extracted file: %v", err)
	}

	fmt.Printf("Extracted: %s\n", file.Name)

	return nil
}
