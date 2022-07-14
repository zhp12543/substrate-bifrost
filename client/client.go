package client

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/zhp12543/substrate-bifrost/models"
	"github.com/zhp12543/substrate-bifrost/utils"
	"github.com/zhp12543/substrate-crypto/ss58"

	"strings"

	gsrc "github.com/zhp12543/substrate-rpc"
	gsClient "github.com/zhp12543/substrate-rpc/client"
	"github.com/zhp12543/substrate-rpc/rpc"
	"github.com/zhp12543/substrate-rpc/types"
	"golang.org/x/crypto/blake2b"
)

type Client struct {
	C                  *gsrc.SubstrateAPI
	Meta               *types.Metadata
	prefix             []byte //币种的前缀
	ChainName          string //链名字
	SpecVersion        int
	TransactionVersion int
	genesisHash        string
	url                string
}

func New(url string) (*Client, error) {
	c := new(Client)
	c.url = url
	var err error

	// 初始化rpc客户端
	c.C, err = gsrc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}
	//检查当前链运行的版本
	err = c.checkRuntimeVersion()
	if err != nil {
		return nil, err
	}
	c.prefix = ss58.BifrostPrefix
	return c, nil
}

func (c *Client) reConnectWs() (*gsrc.SubstrateAPI, error) {
	cl, err := gsClient.Connect(c.url)
	if err != nil {
		return nil, err
	}
	newRPC, err := rpc.NewRPC(cl)
	if err != nil {
		return nil, err
	}
	return &gsrc.SubstrateAPI{
		RPC:    newRPC,
		Client: cl,
	}, nil
}

func (c *Client) checkRuntimeVersion() error {
	v, err := c.C.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		if !strings.Contains(err.Error(), "tls: use of closed connection") {
			return fmt.Errorf("init runtime version error,err=%v", err)
		}
		//	重连处理，这是因为第三方包的问题，所以只能这样处理了了
		cl, err := c.reConnectWs()
		if err != nil {
			return fmt.Errorf("reconnect error: %v", err)
		}
		c.C = cl
		v, err = c.C.RPC.State.GetRuntimeVersionLatest()
		if err != nil {
			return fmt.Errorf("init runtime version error,aleady reconnect,err: %v", err)
		}
	}
	c.TransactionVersion = int(v.TransactionVersion)
	c.ChainName = v.SpecName
	specVersion := int(v.SpecVersion)
	//检查metadata数据是否有升级
	if specVersion != c.SpecVersion {
		c.Meta, err = c.C.RPC.State.GetMetadataLatest()
		if err != nil {
			return fmt.Errorf("init metadata error: %v", err)
		}
		c.SpecVersion = specVersion
	}
	return nil
}

/*
获取创世区块hash
*/
func (c *Client) GetGenesisHash() string {
	if c.genesisHash != "" {
		return c.genesisHash
	}
	hash, err := c.C.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return ""
	}
	c.genesisHash = hash.Hex()
	return hash.Hex()
}

/*
自定义设置prefix，如果启动时加载的prefix是错误的，则需要手动配置prefix
*/
func (c *Client) SetPrefix(prefix []byte) {
	c.prefix = prefix
}

/*
根据height解析block，返回block是否包含交易
*/
func (c *Client) GetBlockByNumber(height int64) (*models.BlockResponse, error) {
	hash, err := c.C.RPC.Chain.GetBlockHash(uint64(height))
	if err != nil {
		return nil, fmt.Errorf("get block hash error:%v,height:%d", err, height)
	}

	return c.GetBlockByHash(hash)
}

/*
根据blockHash解析block，返回block是否包含交易
*/
func (c *Client) GetBlockByHash(bhash types.Hash) (*models.BlockResponse, error) {
	err := c.checkRuntimeVersion()
	if err != nil {
		return nil, err
	}

	sb, err := c.C.RPC.Chain.GetBlock(bhash)
	if err != nil {
		return nil, err
	}

	br := &models.BlockResponse{
		Height:     int64(sb.Block.Header.Number),
		ParentHash: sb.Block.Header.ParentHash.Hex(),
		BlockHash:  sb.Block.Header.ExtrinsicsRoot.Hex(),
		Extrinsic:  sb.Block.Extrinsics,
	}

	return br, nil
}

type parseBlockExtrinsicParams struct {
	from, to, sig, era, txid, fee string
	nonce                         int64
	extrinsicIdx, length          int
}

/*
根据外部交易extrinsic创建txid
*/
func (c *Client) createTxHash(extrinsic string) string {
	data, _ := hex.DecodeString(utils.RemoveHex0x(extrinsic))
	d := blake2b.Sum256(data)
	return "0x" + hex.EncodeToString(d[:])
}

/*
根据地址获取地址的账户信息，包括nonce以及余额等
*/
func (c *Client) GetAccountInfo(address string) (*types.AccountInfo, error) {
	var (
		storage types.StorageKey
		err     error
		pub     []byte
	)
	defer func() {
		if err1 := recover(); err1 != nil {
			err = fmt.Errorf("panic decode event: %v", err1)
		}
	}()
	err = c.checkRuntimeVersion()
	if err != nil {
		return nil, err
	}
	pub, err = ss58.DecodeToPub(address)
	if err != nil {
		return nil, fmt.Errorf("ss58 decode address error: %v", err)
	}
	storage, err = types.CreateStorageKey(c.Meta, "System", "Account", pub, nil)
	if err != nil {
		return nil, fmt.Errorf("create System.Account storage error: %v", err)
	}
	h, _ := c.C.RPC.Chain.GetBlockHashLatest()
	raw, err := c.C.RPC.State.GetStorageRaw(storage, h)
	if err != nil {
		fmt.Println(1111)
	}
	fmt.Println("Key:", storage.Hex())
	fmt.Println(len(*raw))
	fmt.Println(raw.Hex())
	fmt.Println(len(raw.Hex()))

	accountInfo, err := c.C.RPC.State.GetStorageAccountInfo(storage, h)
	if err != nil {
		return nil, fmt.Errorf("get account info error: %v", err)
	}
	return accountInfo, nil
}

/*
获取外部交易extrinsic的手续费
*/
func (c *Client) GetPartialFee(extrinsic, parentHash string) (string, error) {
	if !strings.HasPrefix(extrinsic, "0x") {
		extrinsic = "0x" + extrinsic
	}
	var result map[string]interface{}
	err := c.C.Client.Call(&result, "payment_queryInfo", extrinsic, parentHash)
	if err != nil {
		return "", fmt.Errorf("get payment info error: %v", err)
	}
	if result["partialFee"] == nil {
		return "", errors.New("result partialFee is nil ptr")
	}
	fee, ok := result["partialFee"].(string)
	if !ok {
		return "", fmt.Errorf("partialFee is not string type: %v", result["partialFee"])
	}
	return fee, nil
}
