package account

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"hub/client"
	"hub/common"
	"hub/logger"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

type RpcQueryClient interface {
	Account(address types.AccAddress) (acc exported.Account, err error)
	ChainId() (chainId string, err error)
}

type Account struct {
	ChainId  string
	Key      crypto.PrivKey `json:"-"`
	Coin     types.Coin
	Number   uint64
	Sequence uint64
	Times    int64          `json:"-"`
	NextKey  crypto.PrivKey `json:"-"`
	Receiver types.AccAddress
	Gas      uint64
	Fee      types.Coin
	TxHash   []byte `json:"-"`
}

func NewAccount(cli *client.FastClient, key crypto.PrivKey, fee types.Coin) (*Account, error) {
	chainId, err := cli.ChainId()
	if err != nil {
		return nil, err
	}
	accInfo, err := cli.Account(types.AccAddress(key.PubKey().Address()))
	if err != nil {
		return nil, err
	}
	coin := types.NewCoin(fee.Denom, accInfo.GetCoins().AmountOf(fee.Denom))
	if coin.Amount.IsZero() {
		return nil, fmt.Errorf("%s insufficient account balance", fee.Denom)
	}

	return &Account{
		ChainId:  chainId,
		Key:      key,
		Number:   accInfo.GetAccountNumber(),
		Sequence: accInfo.GetSequence(),
		NextKey:  common.NewPriKey(),
		Receiver: key.PubKey().Address().Bytes(),
		Gas:      100000,
		Fee:      fee,
		Coin:     coin,
	}, nil
}

func (acc *Account) String() string {
	bytes, _ := json.Marshal(acc)
	var data map[string]interface{}
	_ = json.Unmarshal(bytes, &data)
	data["keyAddress"] = types.AccAddress(acc.Key.PubKey().Address().Bytes()).String()
	privKey := acc.Key.(secp256k1.PrivKeySecp256k1)
	data["key"] = hex.EncodeToString(privKey[:])
	marshal, _ := json.MarshalIndent(data, "", "\t")
	return string(marshal)
}

func (acc *Account) GenTransferStdTx(fee types.Coins, msgs ...types.Msg) auth.StdTx {
	sigMsg := auth.StdSignMsg{
		ChainID:       acc.ChainId,
		AccountNumber: acc.Number,
		Sequence:      acc.Sequence,
		Memo:          "fx-jack",
		Msgs:          msgs,
		Fee:           auth.NewStdFee(uint64(len(msgs))*acc.Gas, fee),
	}

	sigBytes, err := acc.Key.Sign(sigMsg.Bytes())
	if err != nil {
		panic(err)
	}

	sig := auth.StdSignature{PubKey: acc.Key.PubKey(), Signature: sigBytes}
	return auth.NewStdTx(sigMsg.Msgs, sigMsg.Fee, []auth.StdSignature{sig}, sigMsg.Memo)
}

func (acc *Account) UpdateAccInfo(cli RpcQueryClient) error {
	accInfo, err := cli.Account(acc.Key.PubKey().Address().Bytes())
	if err != nil {
		return err
	}
	acc.Sequence = accInfo.GetSequence()
	acc.Number = accInfo.GetAccountNumber()
	coin := types.NewCoin(acc.Coin.Denom, accInfo.GetCoins().AmountOf(acc.Coin.Denom))
	if coin.Amount.IsZero() {
		return fmt.Errorf("%s insufficient account balance", acc.Coin.Denom)
	}
	return nil
}

func (acc *Account) BatchDerivedNewAcc(cli *client.FastClient, parallel int64) chan *Account {
	start := time.Now()

	newAccChan := make(chan *Account, parallel)
	newAccChan <- acc

	if parallel <= 1 {
		return newAccChan
	}

	maxParallel := make(chan struct{}, 90)

	/*
		power 2^0 2^1 2^2 2^3 2^4 2^5 2^6 2^7 2^8 2^9 2^10 2^11 2^12 2^13 2^14  2^15
		count 1   2   4   8   16  32  64  128 256 512 1024 2048 4096 8192 16384 32768 ...
	*/
	count := new(big.Int).Exp(big.NewInt(2), big.NewInt(CalculatePower(parallel)), nil)

	for i := int64(1); i < count.Int64()*2; i++ {
		acc := <-newAccChan

		if i >= parallel {
			newAccChan <- acc
			continue
		}

		maxParallel <- struct{}{}

		go func(acc *Account) {
			defer func() { <-maxParallel }()

			// key->nextKey
			transferCoins := types.NewCoins(types.NewCoin(acc.Coin.Denom, acc.Coin.Amount.QuoRaw(2)))
			transferMsg := bank.MsgSend{
				FromAddress: acc.Key.PubKey().Address().Bytes(),
				ToAddress:   acc.NextKey.PubKey().Address().Bytes(),
				Amount:      transferCoins,
			}
			stdTx := acc.GenTransferStdTx(types.NewCoins(acc.Fee), transferMsg)

			_, err := cli.BroadcastStdTxCommitIsOk(stdTx)
			if err != nil {
				logger.L.Errorf("derived new account commit stdtx, err: %s", err.Error())
				os.Exit(1)
			}

			newAccInfo, err := cli.Account(acc.NextKey.PubKey().Address().Bytes())
			if err != nil {
				logger.L.Errorf("query new account info, err: %s", err.Error())
				os.Exit(1)
			}

			newAccount := &Account{
				ChainId:  acc.ChainId,
				Key:      acc.NextKey,
				Coin:     types.NewCoin(acc.Coin.Denom, newAccInfo.GetCoins().AmountOf(acc.Coin.Denom)),
				Number:   newAccInfo.GetAccountNumber(),
				Sequence: newAccInfo.GetSequence(),
				Times:    acc.Times,
				NextKey:  common.NewPriKey(),
				Receiver: acc.Receiver,
				Fee:      acc.Fee,
				Gas:      acc.Gas,
			}

			acc.Sequence = acc.Sequence + 1
			acc.NextKey = common.NewPriKey()
			acc.Coin = types.NewCoin(acc.Coin.Denom, acc.Coin.Amount.QuoRaw(2).Sub(acc.Fee.Amount))

			newAccChan <- newAccount
			newAccChan <- acc
		}(acc)
	}
	for {
		if len(maxParallel) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return newAccChan
}

func (acc *Account) GenAccounts(cli *client.FastClient, number int64) (accounts []*Account, err error) {
	transferAmount := types.NewCoin(acc.Coin.Denom, acc.Coin.Amount.QuoRaw(number).Sub(acc.Fee.Amount))

	var msgs []types.Msg
	for i := int64(0); i < number; i++ {
		newAcc := &Account{
			ChainId:  acc.ChainId,
			Key:      common.NewPriKey(),
			NextKey:  common.NewPriKey(),
			Times:    acc.Times,
			Receiver: acc.Receiver,
			Gas:      acc.Gas,
			Fee:      acc.Fee,
		}
		privateKey := acc.NextKey.(secp256k1.PrivKeySecp256k1)
		logger.L.Infof("Private Key: %s", hex.EncodeToString(privateKey[:]))

		accounts = append(accounts, newAcc)
		msgs = append(msgs, bank.MsgSend{
			FromAddress: acc.Key.PubKey().Address().Bytes(),
			ToAddress:   newAcc.Key.PubKey().Address().Bytes(),
			Amount:      types.NewCoins(transferAmount)},
		)
	}
	_, err = cli.BroadcastStdTxCommitIsOk(acc.GenTransferStdTx(types.NewCoins(acc.Fee), msgs...))
	if err != nil {
		return nil, fmt.Errorf("generate account commit stdTx, err: %s", err.Error())
	}

	for _, account := range accounts {
		newAccInfo, err := cli.Account(account.Key.PubKey().Address().Bytes())
		if err != nil {
			logger.L.Errorf(" query account, err: %s", err.Error())
			return nil, err
		}
		account.Coin = types.NewCoin(acc.Coin.Denom, newAccInfo.GetCoins().AmountOf(acc.Coin.Denom))
		account.Number = newAccInfo.GetAccountNumber()
		account.Sequence = newAccInfo.GetSequence()
	}
	return
}

func WriteAccChanToFile(acc chan *Account) error {
	var list = make([]*Account, len(acc))
	for i := 0; i < len(list); i++ {
		if len(acc) <= 0 {
			break
		}
		list[i] = <-acc
	}

	data, err := json.MarshalIndent(list, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("fx-account.json", data, os.ModePerm)
}

func ReadAccChanToFile() (acc chan *Account, err error) {
	data, err := ioutil.ReadFile("fx-account.json")
	if err != nil {
		return nil, err
	}

	var list = make([]*Account, 0)
	if err = json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	acc = make(chan *Account, len(list))
	for i := 0; i < len(list); i++ {
		acc <- list[i]
	}
	return acc, nil
}

func CalculatePower(parallel int64) int64 {
	if parallel == 1 {
		return 0
	}
	count := int64(2)
	for i := int64(1); i <= 15; i++ {
		if count >= parallel {
			return i
		}
		count = count * 2
	}
	panic(fmt.Sprintf("[%d]", parallel))
}

func (acc *Account) CheckAmount(nodeNum, newAccountNum int64) {
	newAccountNum = newAccountNum * nodeNum
	newAmount := acc.Fee.Amount.Mul(types.NewInt(acc.Times)).Add(types.NewInt(acc.Times))
	totalAmount := newAmount.Mul(types.NewInt(newAccountNum))
	if acc.Coin.Amount.GT(totalAmount) {
		return
	}
	os.Exit(1)
}
