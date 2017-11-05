package selfupdate

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kardianos/osext"
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
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Invalid Response: %v", resp.Status)
	}

	fd, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

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

	execPath, err := osext.Executable()
	if err != nil {
		return err
	}
	dlPath := execPath + ".dl"
	backupPath := execPath + ".bak"

	err = Download(args.DownloadURL, dlPath)
	if err != nil {
		return err
	}

	err = os.Rename(execPath, backupPath)
	if err != nil {
		return err
	}

	err = os.Rename(dlPath, execPath)
	if err != nil {
		os.Rename(backupPath, execPath) // restore backup
		return err
	}

	os.Remove(backupPath) // remove backup

	return nil
}
