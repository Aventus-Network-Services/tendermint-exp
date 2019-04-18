package ticketstore

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/cbergoon/merkletree"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/tendermint/tendermint/abci/types"
	"strconv"
	"strings"
)

const (
	codeTypeOK            	uint32 = 0
	codeTypeEncodingError	uint32 = 1
	codeTypeTicketError  	uint32 = 2
)

var (
	ErrBadAddress = &ticketError{"Ticket must have an address"}
	ErrBadNonce   = &ticketError{"Ticket nonce must increase on resale"}
	ErrBadSignature = &ticketError{"Resale must be signed by the previous owner"}
	ErrTicketNotFound = &ticketError{"Ticket could not be found"}
)

type ticketError struct{ msg string }

func (err ticketError) Error() string { return err.msg }

type state struct {
	size         int64
	height       int64
	rootHash     []byte
	tickets      map[uint64]ticket
	history      map[int64]snapshot
	tempTreeContent  []merkletree.Content
}

type TicketTx struct {
	Id            	uint64  `json:"id"`
	Nonce           uint64  `json:"nonce"`
	Details       	string  `json:"details"`
	OwnerAddr     	string  `json:"ownerAddr"`
	PrevOwnerProof  string  `json:"prevOwnerProof"`
}

type TicketReposnse struct {
	Ticket ticket `json:"ticket"`
	MerkleProof []string `json:"merkleProof"`
}

type TicketStoreApplication struct {
	types.BaseApplication
	state state
}

type ticket struct {
	TicketTx `json:"ticketTx"`
	ChangeHeights []int64 `json:"changeHeights"`
}

type snapshot struct {
	tickets map[uint64]ticket
	tree merkletree.MerkleTree
}

func NewTicketStoreApplication() *TicketStoreApplication {
	return &TicketStoreApplication{state: state{tickets: make(map[uint64]ticket), history: make(map[int64]snapshot)}}
}

func (app *TicketStoreApplication) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{
		Data: fmt.Sprintf("{\"hashes\":%v,\"tickets\":%v}", app.state.height, app.state.size),
		LastBlockHeight: app.state.height,
		LastBlockAppHash: app.state.rootHash}
}

func (app *TicketStoreApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	var ticketTx TicketTx
	err := json.Unmarshal(tx, &ticketTx)

	if err != nil {
		return types.ResponseDeliverTx{
			Code: codeTypeEncodingError,
			Log: fmt.Sprint(err) }
	}

	previousTicket := app.state.tickets[ticketTx.Id]
	err = ticketTx.validate(previousTicket.TicketTx)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: codeTypeTicketError,
			Log: fmt.Sprint(err) }
	}

	app.state.size++
	changeHeights := append(previousTicket.ChangeHeights, app.state.height + 1)
	app.state.tickets[ticketTx.Id] = ticket{ticketTx, changeHeights}
	app.state.tempTreeContent = append(app.state.tempTreeContent, ticketTx)
	return types.ResponseDeliverTx{Code: codeTypeOK}
}

func (app *TicketStoreApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	var ticketTx TicketTx
	err := json.Unmarshal(tx, &ticketTx)

	if err != nil {
		return types.ResponseCheckTx{
			Code: codeTypeEncodingError,
			Log: fmt.Sprint(err) }
	}

	previousTicket := app.state.tickets[ticketTx.Id]
	err = ticketTx.validate(previousTicket.TicketTx)
	if err != nil {
		return types.ResponseCheckTx{
			Code: codeTypeTicketError,
			Log: fmt.Sprint(err) }
	}

	return types.ResponseCheckTx{Code: codeTypeOK}
}

func (app *TicketStoreApplication) Commit() (resp types.ResponseCommit) {
	app.state.height++
	if len(app.state.tempTreeContent) > 0 {
		tree, _ := merkletree.NewTree(app.state.tempTreeContent)
		app.state.rootHash = tree.Root.Hash
		ticketsSnapshot := make(map[uint64]ticket)
		for key, value := range app.state.tickets {
			ticketsSnapshot[key] = value
		}
		app.state.history[app.state.height] = snapshot{ticketsSnapshot, *tree}
		app.state.tempTreeContent = app.state.tempTreeContent[:0]
	}

	return types.ResponseCommit{Data: app.state.rootHash}
}

func (app *TicketStoreApplication) Query(reqQuery types.RequestQuery) types.ResponseQuery {
	switch reqQuery.Path {
	case "hash":
		return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.state.height))}
	case "tx":
		return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.state.size))}
	case "ticket":
		ticket, merkleProof, err := app.state.findTicketAtHeight(string(reqQuery.Data))
		if err != nil {
			return types.ResponseQuery{Log: fmt.Sprintf("%v is not a valid ticket id", reqQuery.Data)}
		}
		response, _ := json.Marshal(TicketReposnse{ticket, merkleProof})
		return types.ResponseQuery{Value: response}
	default:
		return types.ResponseQuery{Log: fmt.Sprintf("Invalid query path. Expected hash or tx, got %v", reqQuery.Path)}
	}
}

func (ticket TicketTx) CalculateHash() ([]byte, error) {
	idBytes := make([]byte,8)
	binary.LittleEndian.PutUint64(idBytes, ticket.Id)
	hash := sha3.SoliditySHA3(
		[]string{"uint256", "uint256", "string", "address", "bytes"},
		[]interface{}{fmt.Sprint(ticket.Id), fmt.Sprint(ticket.Nonce), ticket.Details, ticket.OwnerAddr, ticket.PrevOwnerProof})
	return hash, nil
}

func (ticket TicketTx) Equals(other merkletree.Content) (bool, error) {
	otherTicket, isTicket := other.(TicketTx)
	if isTicket {
		return ticket == otherTicket, nil
	}

	return false, fmt.Errorf("%v is not a ticket", other)
}


func (ticket TicketTx) validate(prevTicket TicketTx) error {
	if ticket.OwnerAddr == "" {
		return ErrBadAddress
	}

	if ticket.Nonce <= prevTicket.Nonce {
		return ErrBadNonce
	}

	if prevTicket.OwnerAddr != "" {
		prevTicketHash, err := prevTicket.CalculateHash()
		if err != nil {
			return err
		}

		signer, err := ticket.getOwnerProofSigner(prevTicketHash)
		if err != nil {
			return err
		}
		if signer != strings.ToLower(prevTicket.OwnerAddr) {
			return ErrBadSignature
		}
	}

	return nil
}

func (ticket TicketTx) getOwnerProofSigner(prevTicketHash []byte) (string, error) {
	bytesProof, err := hexutil.Decode(ticket.PrevOwnerProof)
	if err != nil {
		return "", err
	}

	bytesProof[64] -= 27
	signerPkey, err := crypto.SigToPub(prevTicketHash, bytesProof)
	if err != nil {
		return "", err
	}

	return strings.ToLower(crypto.PubkeyToAddress(*signerPkey).Hex()), nil
}

func (state state) findTicketAtHeight(queryData string) (ticket, []string, error) {
	ticketId, height, err := parseTicketQuery(queryData, state.height)
	if err != nil {
		return ticket{}, nil, err
	}

	lastTicketChange, err := state.tickets[ticketId].findLastChangeBeforeHeight(height)
	if err != nil {
		return ticket{}, nil, err
	}

	snapshot := state.history[lastTicketChange]
	ticket := snapshot.tickets[ticketId]
	merkleProofBytes, _, err := snapshot.tree.GetMerklePath(ticket.TicketTx)
	if err != nil {
		return ticket, nil, err
	}

	merkleProof := make([]string, len(merkleProofBytes))
	for i, v := range merkleProofBytes {
		merkleProof[i] = hexutil.Encode(v)
	}
	return ticket, merkleProof, nil
}

func parseTicketQuery(queryData string, currentHeight int64) (ticketId uint64, height int64, err error) {
	params := strings.Split(queryData, ":")
	ticketId, err = strconv.ParseUint(params[0], 10, 64)
	if err != nil {
		return
	}

	// Height was not provided so assume latest
	if len(params) == 1 {
		height = currentHeight
		return
	}

	height, err = strconv.ParseInt(params[1], 10, 64)
	return
}

func (ticket ticket) findLastChangeBeforeHeight(height int64) (int64, error) {
	for i := len(ticket.ChangeHeights) - 1; i >= 0; i-- {
		if ticket.ChangeHeights[i] <= height {
			return ticket.ChangeHeights[i], nil
		}
	}

	return 0, ErrTicketNotFound
}