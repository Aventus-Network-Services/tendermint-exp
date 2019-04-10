package oddeven

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/tendermint/tendermint/abci/types"
	"strconv"
)

const (
	codeTypeOK            uint32 = 0
	codeTypeEncodingError uint32 = 1
	codeTypeOddNumber     uint32 = 2
)

type state struct {
	size    int64
	height  int64
	appHash []byte
}

type OddEvenApplication struct {
	types.BaseApplication
	state state
}

func (app *OddEvenApplication) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{
		Data: fmt.Sprintf("{\"hashes\":%v,\"txs\":%v}", app.state.height, app.state.size),
		LastBlockHeight: app.state.height,
		LastBlockAppHash: app.state.appHash}
}

func (app *OddEvenApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	txValue, err := parseValue(tx)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: codeTypeEncodingError,
			Log: fmt.Sprintf("%v", err)}
	}

	if txValue % 2 == 1 {
		return types.ResponseDeliverTx{
			Code: codeTypeOddNumber,
			Log:  fmt.Sprintf("%v is not an even number!", txValue)}
	}

	app.state.size++
	return types.ResponseDeliverTx{Code: codeTypeOK}
}

func (app *OddEvenApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	txValue, err := parseValue(tx)
	if err != nil {
		return types.ResponseCheckTx{
			Code: codeTypeEncodingError,
			Log: fmt.Sprintf("%v", err)}
	}

	if txValue % 2 == 1 {
		return types.ResponseCheckTx{
			Code: codeTypeOddNumber,
			Log:  fmt.Sprintf("%v is not an even number!", txValue)}
	}

	return types.ResponseCheckTx{Code: codeTypeOK}
}

func (app *OddEvenApplication) Commit() (resp types.ResponseCommit) {
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.size)
	app.state.height++
	app.state.appHash = appHash

	return types.ResponseCommit{Data: appHash}
}

func (app *OddEvenApplication) Query(reqQuery types.RequestQuery) types.ResponseQuery {
	return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.state.size))}
}

func parseValue(tx []byte) (int, error) {
	txValues := bytes.Split(tx, []byte(":"))
	return strconv.Atoi(string(txValues[0]))
}
