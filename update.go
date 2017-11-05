package selfupdate

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	ErrNoNeedUpdate = errors.New("NoNeedUpdate")
)

func CheckLatestVersion(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Invalid Response: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func Download(url string, savePath string) error {
	fd, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Invalid Response: %v", resp.Status)
	}

	_, err = io.Copy(fd, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

type UpdateArgs struct {
	CurrentVersion string
	VersionURL     string
	DownloadURL    string
}

func Update(args UpdateArgs) error {
	currentVersionInt, err := strconv.ParseInt(strings.TrimSpace(args.CurrentVersion), 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid CurrentVersion %v", err)
	}

	latestVersion, err := CheckLatestVersion(args.VersionURL)
	if err != nil {
		return err
	}

	latestVersionInt, err := strconv.ParseInt(strings.TrimSpace(latestVersion), 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid LatestVersion %v", err)
	}

	if currentVersionInt >= latestVersionInt {
		return ErrNoNeedUpdate
	}

	log.Println("Download the latest version")

	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	dlPath := execPath + ".dl"
	backupPath := execPath + ".bak"

	err = Download(args.DownloadURL, dlPath)
	if err != nil {
		log.Println("Failed to download err:", err)
		return err
	}

	err = os.Rename(execPath, backupPath)
	if err != nil {
		log.Println("Failed to rename exec file for backup")
		return err
	}

	err = os.Rename(dlPath, execPath)
	if err != nil {
		log.Println("Failed to rename download file to exec file")
		os.Rename(backupPath, execPath) // restore backup
		return err
	}

	os.Remove(backupPath) // remove backup

	return nil
}
