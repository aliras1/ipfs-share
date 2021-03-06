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

package interfaces

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/chequebook"

	ethacc "github.com/aliras1/FileTribe/eth/gen/Account"
	"github.com/aliras1/FileTribe/tribecrypto"
)

// IAccount is a mirror to the account data on the blockchain
type IAccount interface {
	ContractAddress() ethcommon.Address
	Contract() *ethacc.Account
	Name() string
	Boxer() tribecrypto.AnonymBoxer
	SetContract(addr ethcommon.Address, backend chequebook.Backend) error
	Save() error
}
