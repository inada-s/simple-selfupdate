package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	ErrNoNeedUpdate       = errors.New("NoNeedUpdate")
	ErrInvalidVersionInfo = errors.New("Invalid Version Json")
)

type version struct {
	Version int    `json:"version"`
	Hash    string `json:"hash"` // hex encoded hash
}

func CheckLatestVersion(url string) (*version, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Invalid Response: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var v = new(version)
	err = json.Unmarshal(body, v)
	if err != nil {
		return nil, err
	}

	// hex decoded
	if len(v.Hash) != sha256.Size*2 {
		return nil, ErrInvalidVersionInfo
	}

	return v, nil
}

// verify sha256 hash of the file and compare it
func VerifySHA256Hash(path string, expected []byte) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()

	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}
	actual := h.Sum(nil)

	if !bytes.Equal(expected, actual) {
		fmt.Errorf("mismatch hash")
	}

	return nil
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

	if currentVersionInt >= int64(latestVersion.Version) {
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

	hash, err := hex.DecodeString(latestVersion.Hash)
	if err != nil {
		return err
	}

	err = VerifySHA256Hash(dlPath, hash)
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
