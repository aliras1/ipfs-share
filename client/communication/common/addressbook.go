package common

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/chequebook"
	"github.com/pkg/errors"
	"ipfs-share/collections"
	"ipfs-share/crypto"
	"ipfs-share/eth/gen/Account"
	ethapp "ipfs-share/eth/gen/Dipfshare"
	"ipfs-share/ipfs"
)

type AddressBook struct {
	accToContactMap *collections.Map
	backend chequebook.Backend
	app *ethapp.Dipfshare
	ipfs ipfs.IIpfs
}

func NewAddressBook(backend chequebook.Backend, app *ethapp.Dipfshare, ipfs ipfs.IIpfs) *AddressBook {
	return &AddressBook{
		backend: 			backend,
		app:				app,
		ipfs:				ipfs,
		accToContactMap:	collections.NewConcurrentMap(),
	}
}

func (a *AddressBook) Get(accAddr ethcommon.Address) (*Contact, error) {
	var c *Contact

	cInt := a.accToContactMap.Get(accAddr)
	if cInt == nil {
		_c, err := a.getContactFromEth(accAddr)
		if err != nil {
			return nil, errors.Wrap(err, "could not get contact")
		}

		c = _c
	} else {
		c = cInt.(*Contact)
	}

	return c, nil
}

func (a *AddressBook) getContactFromEth(accAddr ethcommon.Address) (*Contact, error) {
	acc, err := Account.NewAccount(accAddr, a.backend)
	if err != nil {
		return nil, errors.Wrap(err, "could not create account instance")
	}

	name, err := acc.Name(&bind.CallOpts{Pending:true})
	if err != nil {
		return nil, errors.Wrap(err, "could not get account name")
	}

	ipfsId, err := acc.IpfsId(&bind.CallOpts{Pending:true})
	if err != nil {
		return nil, errors.Wrap(err, "could not get ipfs peer id")
	}

	owner, err := acc.Owner(&bind.CallOpts{Pending:true})
	if err != nil {
		return nil, errors.Wrap(err, "could not get owner")
	}

	contact := NewContact(owner, accAddr, name, ipfsId, crypto.AnonymPublicKey{}, a.ipfs)

	return contact, nil
}
