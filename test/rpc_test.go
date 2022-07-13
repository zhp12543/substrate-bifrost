package test

import (
	"encoding/json"
	"fmt"
	"github.com/zhp12543/substrate-bifrost/client"
	"github.com/zhp12543/substrate-crypto/ss58"
	"testing"
)

func Test_GetBlockByNumber(t *testing.T) {
	c, err := client.New("wss://tanganika.datahighway.com")
	if err != nil {
		t.Fatal(err)
	}
	c.SetPrefix(ss58.DataHighwayPrefix)
	//expand.SetSerDeOptions(false)
	/*
		Ksm: 7834050
	*/
	resp, err := c.GetBlockByNumber(7467638)
	if err != nil {
		t.Fatal(err)
	}

	d, _ := json.Marshal(resp)
	fmt.Println(string(d))
}

func Test_GetAccountInfo(t *testing.T) {
	url := "wss://tanganika.datahighway.com" // wss://kusama-rpc.polkadot.io
	address := "4Kwk17CyU6wi5rHia53r6hrdNyz7THs1EAuca9PbAQRC4siT"

	c, err := client.New(url)
	if err != nil {
		t.Fatal(err)
	}
        c.SetPrefix(ss58.DataHighwayPrefix)
	// 000000000000000001000000000000000047ab56020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
	ai, err := c.GetAccountInfo(address)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := json.Marshal(ai)
	fmt.Println(string(d))
	fmt.Println(ai.Data.Free.String())
}

func Test_GetBlockExtrinsic(t *testing.T) {
	url := "wss://tanganika.datahighway.com" // wss://kusama-rpc.polkadot.io
	c, err := client.New(url)
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.C.RPC.Chain.GetBlockHash(7812476)
	if err != nil {
		t.Fatal(err)
	}
	block, err := c.C.RPC.Chain.GetBlock(h)
	if err != nil {
		t.Fatal(err)
	}
	for _, extrinsic := range block.Block.Extrinsics {
		fmt.Println(extrinsic)
	}
}
