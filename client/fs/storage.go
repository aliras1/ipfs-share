// Copyright (c) 2019 Laszlo Sari
//
// FileTribe is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// FileTribe is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
//

package fs

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"bytes"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/aliras1/FileTribe/client/fs/meta"
	ipfsapi "github.com/aliras1/FileTribe/ipfs"
	"github.com/aliras1/FileTribe/tribecrypto"
	"github.com/aliras1/FileTribe/utils"
)

const (
	CAP_EXT string = ".cap"
)

// Storage is a struct of the directory paths and has
// functions that are responsible for the file level
// functionalities
type Storage struct {
	dataPath        string
	publicPath      string
	publicFilesPath string
	publicForPath   string
	userDataPath    string
	capsPath        string
	origPath        string
	capsGAPath      string // group access caps
	fileRootPath    string
	sharedPath      string
	tmpPath         string
	myFilesPath     string
	ipfsFilesPath   string
	contextDataPath string
}

// NewStorage creates the directory structure
func NewStorage(dataPath string) *Storage {
	glog.Info(dataPath)

	var storage Storage
	storage.dataPath = dataPath + "/data/"
	storage.publicPath = storage.dataPath + "public/"
	storage.publicFilesPath = storage.dataPath + "public/files/"
	storage.publicForPath = storage.dataPath + "public/for/"
	storage.userDataPath = storage.dataPath + "userdata/"
	storage.capsPath = storage.dataPath + "userdata/caps/"
	storage.origPath = storage.dataPath + "userdata/orig/"
	storage.capsGAPath = storage.dataPath + "userdata/caps/GA/"
	storage.fileRootPath = storage.dataPath + "userdata/root/"
	storage.myFilesPath = storage.dataPath + "userdata/root/MyFiles/"
	storage.sharedPath = storage.dataPath + "userdata/shared/"
	storage.tmpPath = storage.dataPath + "userdata/tmp/"
	storage.contextDataPath = storage.dataPath + "userdata/context/"

	os.MkdirAll(storage.dataPath, 0770)
	os.MkdirAll(storage.publicFilesPath, 0770)
	os.MkdirAll(storage.publicForPath, 0770)
	os.MkdirAll(storage.capsPath, 0770)
	os.MkdirAll(storage.origPath, 0770)
	os.MkdirAll(storage.capsGAPath, 0770)
	os.MkdirAll(storage.fileRootPath, 0770)
	os.MkdirAll(storage.myFilesPath, 0770)
	os.MkdirAll(storage.sharedPath, 0770)
	os.MkdirAll(storage.tmpPath, 0770)
	os.MkdirAll(storage.ipfsFilesPath, 0770)
	os.MkdirAll(storage.contextDataPath, 0770)

	return &storage
}

func (storage *Storage) UserFilesPath() string {
	return storage.fileRootPath
}

func (storage *Storage) CopyFileIntoPublicDir(filePath string) error {
	publicFilePath := storage.publicFilesPath + "/" + path.Base(filePath)
	return utils.CopyFile(filePath, publicFilePath)
}

func (storage *Storage) CopyFileIntoMyFiles(filePath string) (string, error) {
	newFilePath := storage.myFilesPath + "/" + path.Base(filePath)
	return newFilePath, utils.CopyFile(filePath, newFilePath)
}

func (storage *Storage) CopyFileIntoGroupFiles(filePath, groupName string) error {
	groupFilesPath := storage.fileRootPath + "/" + groupName
	os.Mkdir(groupFilesPath, 0770)
	newFilePath := groupFilesPath + "/" + path.Base(filePath)
	return utils.CopyFile(filePath, newFilePath)
}

func (storage *Storage) SaveAccountData(data []byte) error {
	path := storage.userDataPath + "account.dat"

	if err := utils.WriteFile(path, data); err != nil {
		return errors.Wrapf(err, "could not write to file: %s", path)
	}

	return nil
}

func (storage *Storage) LoadAccountData() ([]byte, error) {
	path := storage.userDataPath + "account.dat"

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file: %s", path)
	}

	return data, nil
}

// +------------------------------+
// |   Group specific functions   |
// +------------------------------+

// Gets all the locally stored group access capabilities from
// directory data/userdata/caps/GA/
func (storage *Storage) GetGroupMetas() ([]*meta.GroupMeta, error) {
	var groupMetas []*meta.GroupMeta
	// read groupMetas from caps and try to refresh them
	groupCapFiles, err := ioutil.ReadDir(storage.capsGAPath)
	if err != nil {
		return groupMetas, err
	}

	for _, groupCapFile := range groupCapFiles {
		if groupCapFile.IsDir() {
			continue // do not care about directories
		}

		filePath := storage.capsGAPath + "/" + groupCapFile.Name()
		capBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			glog.Warning("could not read file '%s': Storage.GetGroupMetas: %s", filePath, err)
			continue
		}

		var groupMeta meta.GroupMeta
		if err := json.Unmarshal(capBytes, &groupMeta); err != nil {
			glog.Warning("could not unmarshal group groupMeta: Storage.GetGroupMetas: %s", err)
			continue
		}

		groupMeta.Boxer.RNG = rand.Reader
		groupMetas = append(groupMetas, &groupMeta)
	}

	return groupMetas, nil
}

func (storage *Storage) GetGroupFileMetas(groupAddress string) ([]*meta.FileMeta, error) {
	var fileMetas []*meta.FileMeta

	baseDir := storage.GroupFileCapDir(groupAddress)
	metaFiles, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return fileMetas, err
	}

	for _, metaFile := range metaFiles {
		if metaFile.IsDir() {
			continue // do not care about directories
		}

		filePath := baseDir + "/" + metaFile.Name()
		metaBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			glog.Warning("could not read file '%s': Storage.GetGroupFileMetas: %s", filePath, err)
			continue
		}

		var fileMeta meta.FileMeta
		if err := json.Unmarshal(metaBytes, &fileMeta); err != nil {
			glog.Warning("could not unmarshal group fileMeta: Storage.GetGroupFileMetas: %s", err)
			continue
		}

		fileMetas = append(fileMetas, &fileMeta)
	}

	return fileMetas, nil
}

func (storage *Storage) SaveGroupAccessCap(cap *meta.GroupMeta) error {
	capJson, err := json.Marshal(cap)
	if err != nil {
		return errors.Wrap(err, "could not marshal group access capability")
	}

	path := storage.GroupAccessCapDir() + cap.Address.String() + CAP_EXT
	if err := utils.WriteFile(path, capJson); err != nil {
		return errors.Wrap(err, "could not write group cap file")
	}

	return nil
}

func (storage *Storage) GroupAccessCapDir() string {
	return storage.capsGAPath
}

func (storage *Storage) GroupFileCapDir(id string) string {
	return storage.capsPath + id + "/"
}

func (storage *Storage) GroupFileOrigDir(id string) string {
	return storage.origPath + id + "/"
}

func (storage *Storage) GroupFileDataDir(id string) string {
	return storage.fileRootPath + id + "/"
}

// Creates the directory structure needed by a group
func (storage *Storage) MakeGroupDir(id string) {
	os.MkdirAll(storage.capsPath + id, 0770)
	os.MkdirAll(storage.fileRootPath + id, 0770)
	os.MkdirAll(storage.origPath+ id, 0770)
}

func (storage *Storage) DownloadTmpFile(ipfsHash string, ipfs ipfsapi.IIpfs) (string, error) {
	filePath := storage.tmpPath + "/" + ipfsHash
	if err := ipfs.Get(ipfsHash, filePath); err != nil {
		return "", fmt.Errorf("could not ipfs get into tmp path '%s': Storage.downloadToTmp: %s", filePath, err)
	}
	return filePath, nil
}

func (storage *Storage) DownloadAndDecryptWithSymmetricKey(boxer tribecrypto.SymmetricKey, ipfsHash string, ipfs ipfsapi.IIpfs) ([]byte, error) {
	path := storage.tmpPath + ipfsHash
	if err := ipfs.Get(ipfsHash, path); err != nil {
		return nil, errors.Wrapf(err, "could not ipfs get ipfs hash %s", ipfsHash)
	}

	encData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read tmp file: %s", path)
	}
	defer func() {
		if err := os.Remove(path); err != nil {
			glog.Warningf("could not remove tmp file %s", path)
		}
	}()

	data, ok := boxer.BoxOpen(encData)
	if !ok {
		return nil, errors.New("could not decrypt shared group dir")
	}

	return data, nil
}

func (storage *Storage) DownloadAndDecryptWithFileBoxer(boxer tribecrypto.FileBoxer, ipfsHash string, ipfs ipfsapi.IIpfs) ([]byte, error) {
	tmpFilePath, err := storage.DownloadTmpFile(ipfsHash, ipfs)
	if err != nil {
		return nil, errors.Wrapf(err, "could not ipfs get '%s'", ipfsHash)
	}

	encReader, err := os.Open(tmpFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "download err: could not read file '%s'", tmpFilePath)
	}

	diffBuf := new(bytes.Buffer)
	err = boxer.Open(encReader, diffBuf)
	defer func() {
		if err := encReader.Close(); err != nil {
			glog.Warningf("download err: could not close tmp file '%s': %s", tmpFilePath, err)
		}
		if err := os.Remove(tmpFilePath); err != nil {
			glog.Warningf("download err: could not delete tmp file '%s': %s", tmpFilePath, err)
		}
	}()

	if err != nil {
		return nil, errors.Wrap(err, "download err: could not decrypt file dif")
	}

	return diffBuf.Bytes(), nil
}
