package client

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	fs "ipfs-share/client/filestorage"
	"ipfs-share/crypto"
	"ipfs-share/ipfs"
	nw "ipfs-share/network"
)

type SignedBy struct {
	Username  string `json:"username"`
	Signature []byte `json:"signature"`
}

type Operation struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type Transaction struct {
	PrevState []byte     `json:"prev_state"`
	State     []byte     `json:"state"`
	Operation Operation  `json:"operation"`
	SignedBy  []SignedBy `json:"signed_by"`
}

func (t *Transaction) Bytes() []byte {
	var transactionBytes []byte
	transactionBytes = append(transactionBytes, t.PrevState...)
	transactionBytes = append(transactionBytes, t.State...)
	transactionBytes = append(transactionBytes, []byte(t.Operation.Type)...)
	transactionBytes = append(transactionBytes, []byte(t.Operation.Data)...)
	return transactionBytes
}

type Member struct {
	Name      string                  `json:"name"`
	VerifyKey crypto.PublicSigningKey `json:"-"`
}

type MemberList struct {
	List []Member
}

func NewMemberList() *MemberList {
	return &MemberList{ List:[]Member{}}
}

func (ml *MemberList) Length() int {
	return len(ml.List)
}

func (ml *MemberList) Hash() [32]byte {
	var data []byte
	for _, member := range ml.List {
		data = append(data, []byte(member.Name)...)
	}
	return sha256.Sum256(data)
}

func (ml *MemberList) Append(user string, network *nw.Network) *MemberList {
	verifyKey, err := network.GetUserVerifyKey(user)
	if err != nil {
		log.Printf("could not get user verify key: MemberList.Append: %s", err)
		return ml
	}
	newList := make([]Member, len(ml.List))
	copy(newList, ml.List)
	newList = append(newList, Member{user, verifyKey})
	return &MemberList{List: newList}
}

func (ml *MemberList) Get(user string) *Member {
	for i := 0; i < ml.Length(); i++ {
		if strings.Compare(ml.List[i].Name, user) == 0 {
			return &ml.List[i]
		}
	}
	return nil
}

type GroupContext struct {
	User         *User
	Group        *Group
	Repo         []*fs.FilePTP
	Members      *MemberList
	Synchronizer *Synchronizer
	Network      *nw.Network
	IPFS         *ipfs.IPFS
	Storage      *fs.Storage
}

func NewGroupContext(group *Group, user *User, network *nw.Network, ipfs *ipfs.IPFS, storage *fs.Storage) (*GroupContext, error) {
	members := NewMemberList()
	memberStrings, err := network.GetGroupMembers(group.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get group members of '%s': NewGroupContext: %s", group.Name, err)
	}
	for _, member := range memberStrings {
		members = members.Append(member, network)
	}
	groupContext := &GroupContext{
		User:         user,
		Group:        group,
		Repo:         nil,
		Members:      members,
		Synchronizer: nil,
		Network:      network,
		IPFS:         ipfs,
		Storage:      storage,
	}
	groupContext.Synchronizer = NewSynchronizer(groupContext)
	return groupContext, nil
}

func NewGroupContextFromCAP(cap *fs.GroupAccessCAP, user *User, network *nw.Network, ipfs *ipfs.IPFS, storage *fs.Storage) (*GroupContext, error) {
	group := &Group{
		Name:  cap.GroupName,
		Boxer: cap.Boxer,
	}
	gc, err := NewGroupContext(group, user, network, ipfs, storage)
	if err != nil {
		return nil, fmt.Errorf("could not create group context: NewGroupContextFromCAP: %s", err)
	}
	return gc, nil
}

func (gc *GroupContext) CalculateState() []byte {
	state := gc.Members.Hash()
	return state[:]
}

func (gc *GroupContext) GetState() ([]byte, error) {
	state, err := gc.Network.GetGroupState(gc.Group.Name)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve state from network: GroupContext.GetState: %s", err)
	}
	return state, nil
}

func (gc *GroupContext) Invite(newMember string) error {
	prevHash := gc.Members.Hash()
	newMembers := gc.Members.Append(newMember, gc.Network)
	newHash := newMembers.Hash()

	operation := Operation{
		Type: "INVITE",
		Data: gc.User.Name + " " + newMember,
	}

	transaction := Transaction{
		PrevState: prevHash[:],
		State:     newHash[:],
		Operation: operation,
		SignedBy:  []SignedBy{},
	}
	// fork down the collection of signatures for the operation
	go gc.Synchronizer.CollectApprovals(&transaction)

	// send out the proposed transaction to be signed
	transactionBytes, err := json.Marshal(transaction)
	if err != nil {
		return fmt.Errorf("could not marshal transaction: GroupContext.Invite: %s", err)
	}
	signedTransactionBytes := gc.User.Signer.SecretKey.Sign(nil, transactionBytes)
	groupMsg := GroupMessage{
		Type: "PROPOSAL",
		From: gc.User.Name,
		Data: signedTransactionBytes,
	}
	groupMsgBytes, err := json.Marshal(groupMsg)
	if err != nil {
		return fmt.Errorf("could not marshal group message: GroupContext.Invite: %s", err)
	}
	if err := gc.sendToAll(groupMsgBytes); err != nil {
		return fmt.Errorf("could not send group message: GroupContext.Invite: %s", err)
	}
	return nil
}

func (gc *GroupContext) Save() error {
	return fmt.Errorf("not implemented: GroupContext.Save")
}

// Sends pubsub messages to all members of the group
func (gc *GroupContext) sendToAll(data []byte) error {
	encGroupMsg := gc.Group.Boxer.BoxSeal(data)
	if err := gc.IPFS.PubsubPublish(gc.Group.Name, encGroupMsg); err != nil {
		return fmt.Errorf("could not pubsub publish: GroupContext.sendToAll: %s", err)
	}
	return nil
}

// Pulls from others the given group meta data
func (gc *GroupContext) PullGroupData(data string) error {
	return fmt.Errorf("not implemented: GroupContext.PullGroupData")
}

// Loads the locally available group meta data (stored in the
// data/public/for/GROUP/ directory).
func (gc *GroupContext) LoadGroupData(data string) error {
	return fmt.Errorf("not implemented GroupContext.LoadGroupData")
}
