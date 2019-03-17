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

package clients

import (
	"math/rand"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	comcommon "github.com/aliras1/FileTribe/client/communication/common"
	"github.com/aliras1/FileTribe/client/communication/sessions/common"
	"github.com/aliras1/FileTribe/tribecrypto"
)

type GetGroupDataSessionClient struct {
	sessionId          uint32
	state              uint8
	receiver           *comcommon.Contact
	groupDataMsg comcommon.GroupDataMessage

	sender          ethcommon.Address
	onSessionClosed common.SessionClosedCallback
	signer          *tribecrypto.Signer

	lock 				sync.RWMutex
	stop			    chan bool
	error				error
	onSuccessCallback   common.OnGetGroupKeySuccessCallback
}

func (session *GetGroupDataSessionClient) Error() error {
	return session.error
}

func (session *GetGroupDataSessionClient) close() {
	session.state = common.EndOfSession
	session.onSessionClosed(session)
}

func (session *GetGroupDataSessionClient) Abort() {
	if !session.IsAlive() {
		return
	}

	session.close()
}

func (session *GetGroupDataSessionClient) State() uint8 {
	session.lock.RLock()
	defer session.lock.RUnlock()

	return session.state
}

func (session *GetGroupDataSessionClient) Id() uint32 {
	return session.sessionId
}

func (session *GetGroupDataSessionClient) IsAlive() bool {
	session.lock.RLock()
	defer session.lock.RUnlock()

	return session.state == common.EndOfSession
}

func (session *GetGroupDataSessionClient) Run() {
	session.NextState(nil, nil)
}

func (session *GetGroupDataSessionClient) NextState(contact *comcommon.Contact, data []byte) {
	session.lock.Lock()
	defer session.lock.Unlock()

	switch session.state {
	case 0:
		{
			glog.Infof("client [%d] {%s} [0] --> %s", session.sessionId, session.sender.String(), session.receiver.AccAddr.String())
			payload, err := session.groupDataMsg.Encode()
			if err != nil {
				session.error = errors.Wrap(err, "could not encoder message payload")
				session.close()
				return
			}

			glog.Infof("client %d [0][0]", session.sessionId)

			msg, err := comcommon.NewMessage(
				session.sender,
				comcommon.GetGroupData,
				session.sessionId,
				payload,
				session.signer,
			)
			if err != nil {
				session.error = errors.Wrap(err, "could not create message")
				session.close()
				return
			}
			glog.Infof("client %d [0][1]", session.sessionId)
			encMsg, err := msg.Encode()
			if err != nil {
				session.error = errors.Wrap(err, "could not encode message")
				session.close()
				return
			}
			glog.Infof("client %d [0][2]", session.sessionId)
			if err := session.receiver.Send(encMsg); err != nil {
				session.error = errors.Wrap(err, "could not send message")
				session.close()
				return
			}
			glog.Infof("client %d [0][3]", session.sessionId)

			session.state = 1

			return
		}
		// Got the challenge
	case 1:
		{
			glog.Infof("client %s [1] --> %s", session.sender.String(), session.receiver.AccAddr.String())
			sig, err := session.signer.Sign(data)
			if err != nil {
				session.error = errors.Wrap(err, "could not sign challenge")
				session.close()
			}

			msg, err := comcommon.NewMessage(
				session.sender,
				comcommon.GetGroupData,
				session.sessionId,
				sig,
				session.signer,
			)
			if err != nil {
				session.error = errors.Wrap(err, "could not create message")
				session.close()
				return
			}

			encMsg, err := msg.Encode()
			if err != nil {
				session.error = errors.Wrap(err, "could not encode Message")
				session.close()
				return
			}

			if err := session.receiver.Send(encMsg); err != nil {
				session.error = errors.Wrap(err, "could not send message")
				session.close()
				return
			}

			session.state = 2

			return
		}
	case 2:
		{
			glog.Infof("client [2] --> %s", session.receiver.AccAddr.String())
			key, err := tribecrypto.DecodeSymmetricKey(data)
			if err != nil {
				session.error = errors.Wrap(err, "could not decode group key")
				session.close()
				return
			}

			switch session.groupDataMsg.Data {
			case comcommon.GetGroupKey:
				session.onSuccessCallback(session.groupDataMsg.Group, *key)

			case comcommon.GetProposedGroupKey:
				session.onSuccessCallback(ethcommon.BytesToAddress(session.groupDataMsg.Payload), *key)
			}

			session.close()
			return
		}

	default:
		{
			glog.Errorf("session ended")
		}
	}
}

func NewGetGroupDataSessionClient(
	requestedGroupData comcommon.GroupData,
	groupAddr ethcommon.Address,
	groupMsgPayload []byte,
	contact *comcommon.Contact,
	sender ethcommon.Address,
	signer *tribecrypto.Signer,
	onSessionClosed common.SessionClosedCallback,
	onSuccess common.OnGetGroupKeySuccessCallback,
) *GetGroupDataSessionClient {

	groupDataMsg := comcommon.GroupDataMessage{
		Group: groupAddr,
		Data: requestedGroupData,
		Payload: groupMsgPayload,
	}

	rand.Seed(time.Now().UTC().UnixNano())
	return &GetGroupDataSessionClient{
		sessionId:         	rand.Uint32(),
		groupDataMsg:		groupDataMsg,
		receiver:          	contact,
		state:             	0,
		sender:            	sender,
		signer:            	signer,
		onSessionClosed:   	onSessionClosed,
		stop:              	make(chan bool),
		onSuccessCallback: 	onSuccess,
	}
}