package oddeven

import (
	"encoding/binary"
	"fmt"
	"github.com/tendermint/tendermint/abci/types"
)

const (
	codeTypeOK            uint32 = 0
	codeTypeEncodingError uint32 = 1
	codeTypeOddNumber     uint32 = 2
)

type state struct {
	size    uint64
	height  uint64
}

type OddEvenApplication struct {
	types.BaseApplication
	state state
}

func (app *OddEvenApplication) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{Data: fmt.Sprintf("{\"hashes\":%v,\"txs\":%v}", app.state.height, app.state.size)}
}

func (app *OddEvenApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	if len(tx) > 8 {
		return types.ResponseDeliverTx{
			Code: codeTypeEncodingError,
			Log:  fmt.Sprintf("Max tx size is 8 bytes, got %d", len(tx))}
	}

	tx8 := make([]byte, 8)
	copy(tx8[len(tx8)-len(tx):], tx)
	txValue := binary.BigEndian.Uint64(tx8)

	if txValue % 2 == 1 {
		return types.ResponseDeliverTx{
			Code: codeTypeOddNumber,
			Log:  fmt.Sprintf("%v is not an even number!", txValue)}
	}

	app.state.size++
	return types.ResponseDeliverTx{Code: codeTypeOK}
}

func (app *OddEvenApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	if len(tx) > 8 {
		return types.ResponseCheckTx{
			Code: codeTypeEncodingError,
			Log:  fmt.Sprintf("Max tx size is 8 bytes, got %d", len(tx))}
	}

	tx8 := make([]byte, 8)
	copy(tx8[len(tx8)-len(tx):], tx)
	txValue := binary.BigEndian.Uint64(tx8)

	if txValue % 2 == 1 {
		return types.ResponseCheckTx{
			Code: codeTypeOddNumber,
			Log:  fmt.Sprintf("%v is not an even number!", txValue)}
	}

	return types.ResponseCheckTx{Code: codeTypeOK}
}

func (app *OddEvenApplication) Commit() (resp types.ResponseCommit) {
	appHash := make([]byte, 8)
	binary.PutUvarint(appHash, app.state.size)
	app.state.height++
	return types.ResponseCommit{Data: appHash}
}

func (app *OddEvenApplication) Query(reqQuery types.RequestQuery) types.ResponseQuery {
	return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.state.size))}
}
