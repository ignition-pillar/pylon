package main

import (
	"log"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignition-pillar/go-zdk/client"
	"github.com/ignition-pillar/go-zdk/utils"
	"github.com/ignition-pillar/go-zdk/utils/template"
	signer "github.com/ignition-pillar/go-zdk/wallet"
	"github.com/ignition-pillar/go-zdk/zdk"
	"github.com/tyler-smith/go-bip39"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/wallet"
)

const (
	Decimals = 100000000
)

func connect(url string, chainId int) (*zdk.Zdk, error) {
	rpc, err := client.NewClient(url, client.ChainIdentifier(uint64(chainId)))
	if err != nil {
		return nil, err
	}
	z := zdk.NewZdk(rpc)
	return z, nil
}

type FaucetRequest struct {
	Address types.Address `json:"address"`
}

func main() {

	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)

	ks := &wallet.KeyStore{
		Entropy:  entropy,
		Seed:     bip39.NewSeed(mnemonic, ""),
		Mnemonic: mnemonic,
	}
	_, keyPair, _ := ks.DeriveForIndexPath(0)
	ks.BaseAddress = keyPair.Address
	log.Println(ks.BaseAddress)
	kp := signer.NewSigner(keyPair)

	z, _ := connect("ws://127.0.0.1:35998", 321)

	r := gin.Default()
	r.GET("/frontierMomentum", func(c *gin.Context) {
		m, err := z.Ledger.GetFrontierMomentum()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"hash":   m.Hash,
			"height": m.Height,
		})
	})

	r.POST("/faucet", func(c *gin.Context) {
		fr := FaucetRequest{}
		if err := c.BindJSON(&fr); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		tx := template.Send(1, 321, fr.Address, types.ZnnTokenStandard, big.NewInt(10*Decimals), []byte{})
		block, err := utils.Send(z, tx, kp, true)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = z.Ledger.PublishRawTransaction(block)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusAccepted, &fr)
	})

	r.Run()
}
