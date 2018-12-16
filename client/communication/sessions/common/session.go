package common

import (
	"math"

	"ipfs-share/client/communication/common"
)


const EndOfSession = math.MaxUint8

type ISession interface {
	Id() uint32
	IsAlive() bool
	Abort()
	NextState(contact *common.Contact, data []byte)
	State() uint8
	Run()
	Error() error
}
