package filestorage

import (
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"path"

	"golang.org/x/crypto/ed25519"

	"fmt"
	"ipfs-share/crypto"
	"ipfs-share/ipfs"
	nw "ipfs-share/network"
	"os"
	"strings"
)

type File interface {
	Share()
	//NewFileFromCAP(cap CAP, )
}

type FilePTP struct {
	Name       string                  `json:"name"`
	Owner      string                  `json:"owner"`
	IPFSHash   string                  `json:"ipfs_hash"`
	IPNSPath   string                  `json:"ipns_path"`
	Path       string                  `json:"path"`
	SharedWith []string                `json:"shared_with"`
	WAccess    []string                `json:"w_access"`
	VerifyKey  crypto.PublicSigningKey `json:"verify_key"`
	WriteKey   crypto.SecretSigningKey `json:"write_key"`
	Own        bool                    `json:"own"` // current user owns the file?
	// it could be a good idea to hardwire Owner into the file data
	// as well and validate it...
}

// New FilePTP object from local data
func NewFile(filePath string) (*FilePTP, error) {
	bytesFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s': NewFile: %s", filePath, err)
	}
	var file FilePTP
	if err := json.Unmarshal(bytesFile, &file); err != nil {
		fmt.Errorf("could not unmarshal file '%s': NewFile: %s", filePath, err)
	}
	return &file, nil
}

func NewFileFromCAP(cap *ReadCAP, storage *Storage, ipfs *ipfs.IPFS) (*FilePTP, error) {
	filePath := storage.fileRootPath + "/" + cap.Owner + "/" + cap.FileName
	ipfsHash, err := ipfs.Resolve(cap.IPNSPath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve ipns address: NewFileFromCAP: %s", err)
	}
	fmt.Println(ipfsHash)
	file := FilePTP{cap.FileName,
		cap.Owner,
		ipfsHash,
		cap.IPNSPath,
		filePath,
		[]string{},
		[]string{},
		cap.VerifyKey,
		crypto.SecretSigningKey{},
		false,
	}
	fmt.Println(file)
	if err := file.download(storage, ipfs); err != nil {
		return nil, fmt.Errorf("could not download file '%s': NewFileFromCAP: %s", cap.FileName, err)
	}
	if err := file.save(storage); err != nil {
		return nil, fmt.Errorf("could not save file '%s': NewFileFromCAP: %s", file.Name, err)
	}
	return &file, nil
}

// Downloads and verifies the file from IPFS
func (f *FilePTP) download(storage *Storage, ipfs *ipfs.IPFS) error {
	tmpFilePath := storage.tmpPath + "/" + path.Base(f.Name)
	err := ipfs.Get(tmpFilePath, f.IPFSHash)
	if err != nil {
		return fmt.Errorf("could not ipfs get '%s': FilePTP.download: %s", f.IPFSHash, err)
	}
	bytesSignedFile, err := ioutil.ReadFile(tmpFilePath)
	if err != nil {
		return fmt.Errorf("could not read file '%s': FilePTP.download: %s", tmpFilePath, err)
	}
	os.Remove(tmpFilePath)
	// make a directory to the owner
	dirPath := storage.fileRootPath + "/" + f.Owner
	os.MkdirAll(dirPath, 0770)

	bytesRawFile, ok := f.VerifyKey.Open(nil, bytesSignedFile)
	if !ok {
		return fmt.Errorf("could not verify file '%s': FilePTP.download: %s", f.Name, err)
	}
	filePath := dirPath + "/" + f.Name
	if err := WriteFile(filePath, bytesRawFile); err != nil {
		return fmt.Errorf("could not write file '%s': FilePTP.download: %s", f.Name, err)
	}
	return nil
}

func (f *FilePTP) save(storage *Storage) error {
	path := storage.capsPath + "/" + f.Name
	jsonBytes, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("could not marshal file '%s': FilePTP.save: %s", f.Name, err)
	}
	if err := WriteFile(path, jsonBytes); err != nil {
		return fmt.Errorf("could not write file '%s': FilePTP.save: %s", path, err)
	}
	return nil
}

// Create a new shared file object from a local file
func NewSharedFile(filePath, owner string, storage *Storage, ipfs *ipfs.IPFS) (*FilePTP, error) {
	newFilePath, err := storage.CopyFileIntoMyFiles(filePath)
	if err != nil {
		return nil, err
	}

	vk, wk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	verifyKey := crypto.PublicSigningKey(vk)
	writeKey := crypto.SecretSigningKey(wk)

	ipfsID, err := ipfs.ID()
	if err != nil {
		return nil, fmt.Errorf("could not get ipfs id: NewSharedFile: %s", err)
	}
	fileName := path.Base(filePath)
	file := &FilePTP{fileName,
		owner,
		"",
		"/ipns/" + ipfsID.ID + "/files/" + fileName,
		newFilePath,
		[]string{},
		[]string{},
		verifyKey,
		writeKey,
		true,
	}
	if err := file.signAndAddToIPFS(storage, ipfs); err != nil {
		return nil, fmt.Errorf("could not sign and add file '%s' to ipfs: NewSharedFile: %s", fileName, err)
	}
	if err := file.save(storage); err != nil {
		return nil, fmt.Errorf("could not save file '%s': NewSharedFile: %s", file.Name, err)
	}
	return file, nil
}

// Signs the files with the Write key and then the function adds
// it to IPFS. The function returns with the with the IPFS hash
// of the file
func (f *FilePTP) signAndAddToIPFS(storage *Storage, ipfs *ipfs.IPFS) error {
	publicFilePath := storage.publicFilesPath + "/" + f.Name
	bytesFile, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return fmt.Errorf("could not read file '%s': FilePTP.signAndAddToIPFS: %s", f.Name, err)
	}
	if f.WriteKey == nil {
		return fmt.Errorf("no write key found in file '%s': FilePTP.signAndAddToIPFS", f.Name)
	}
	signedFile := f.WriteKey.Sign(nil, bytesFile)
	if err := WriteFile(publicFilePath, signedFile); err != nil {
		return fmt.Errorf("could not write signed file '%s': FilePTP.signAndAddToIPFS: %s", f.Name, err)
	}
	merkleNode, err := ipfs.AddFile(publicFilePath)
	if err != nil {
		return fmt.Errorf("could not add file '%s' to ipfs: FilePTP.signAndAddToIPFS: %s", f.Name, err)
	}
	f.IPFSHash = merkleNode.Hash
	return nil
}

// Share file with a set of users, described by shareWith. Encrypted
// capabilities are made and copied in the corresponding 'public/for/'
// directories. The 'public' directory is re-published into IPNS. After
// that, notification messages are sent out.
func (f *FilePTP) Share(shareWith []string, boxer *crypto.BoxingKeyPair, storage *Storage, network *nw.Network, ipfs *ipfs.IPFS) error {
	var newUsers []string
	for _, user := range shareWith {
		// add to share list
		f.SharedWith = append(f.SharedWith, user)
		// make new capability into for_X directory
		if err := CreateFileReadCAPForUser(f, user, f.IPNSPath, boxer, storage, network); err != nil {
			return fmt.Errorf("could not create CAP for file '%s' for user '%s': FilePTP.Share: %s", f.Name, user, err)
		}
		// NOTE: we cannot send notification messages here because
		// from efficiency considerations /public directory will be
		// published just once, with all the new CAPs in it
		newUsers = append(newUsers, user)
	}
	f.save(storage)
	err := storage.PublishPublicDir(ipfs)
	if err != nil {
		return fmt.Errorf("could not publish public dir: Share: %s", err)
	}
	// send share messages
	for _, user := range newUsers {
		err = network.SendMessage(f.Owner, user, "PTP READ CAP", path.Base(f.Name)+".json")
		if err != nil {
			return fmt.Errorf("could not send 'PTP READ CAP' message to user '%s': Share: %s", user, err)
		}
	}
	return nil
}

func (f *FilePTP) Refresh(storage *Storage, ipfs *ipfs.IPFS) error {
	if f.Own {
		return nil
	}
	newIPFSHash, err := ipfs.Resolve(f.IPNSPath)
	if err != nil {
		return fmt.Errorf("could not resolve ipns path '%s': FilePTP.Refresh: %s", f.IPNSPath, err)
	}
	if strings.Compare(newIPFSHash, f.IPFSHash) != 0 {
		f.IPFSHash = newIPFSHash
		if err := f.download(storage, ipfs); err != nil {
			return fmt.Errorf("could not download file '%s': FilePTP.Refresh: %s", f.Name, err)
		}
	}
	return nil
}
