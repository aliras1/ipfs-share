package fs

import (
	"github.com/sergi/go-diff/diffmatchpatch"
	"encoding/json"
	"bytes"
	"ipfs-share/crypto"
	"io"
	"github.com/pkg/errors"
)

type DiffNode struct {
	Hash []byte
	Diff []diffmatchpatch.Diff
	Next string
}

func (diff *DiffNode) Encode() ([]byte, error) {
	data, err := json.Marshal(diff)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeDiffNode(data []byte) (*DiffNode, error) {
	var diff DiffNode
	if err := json.Unmarshal(data, &diff); err != nil {
		return nil, err
	}
	return &diff, nil
}

func (diff *DiffNode) Encrypt(boxer crypto.FileBoxer) (io.Reader, error) {
	data, err := diff.Encode()
	if err != nil {
		return nil, errors.Wrap(err, "could not encode diff node")
	}
	encData, err := boxer.Seal(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt data")
	}

	return encData, nil
}