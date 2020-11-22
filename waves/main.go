package main

import (
	"context"
	"flag"
	"fmt"
	wavesCrypto "github.com/wavesplatform/go-lib-crypto"
	"time"

	"rh_tests/contracts"
	"rh_tests/deployer"

	"github.com/wavesplatform/gowaves/pkg/proto"

	"github.com/wavesplatform/gowaves/pkg/crypto"

	wavesClient "github.com/wavesplatform/gowaves/pkg/client"
	"rh_tests/helpers"
)

const (
	DeployOperation = "deploy"
)

var operation, configFile string

func init() {
	flag.StringVar(&operation, "operation", DeployOperation, "What action to perform")
	flag.StringVar(&configFile, "config", "config.json", "Config file to read from")
	flag.Parse()
}

func main() {
	var err error

	switch operation {
	case DeployOperation:
		_, err = Deploy()
	}

	if err != nil {
		fmt.Printf("Error occured: %v \n", err)
	}
}

func Deploy() (*DeploymentConfig, error) {
	const (
		BftValue    = 3
		Wavelet     = 1e8
	)

	var testConfig DeploymentConfig
	testConfig.Ctx = context.Background()

	cfg, err := LoadDeploymentConfig(configFile)
	if err != nil {
		return nil, err
	}

	wClient, err := wavesClient.NewClient(wavesClient.Options{ApiKey: "", BaseUrl: cfg.NodeUrl})
	if err != nil {
		return nil, err
	}
	testConfig.Client = wClient
	testConfig.Helper = helpers.NewClientHelper(testConfig.Client)

	testConfig.Consuls = cfg.ConsulsAddressList

	testConfig.Gravity, err = GenerateAddressFromSeed(cfg.ChainId, cfg.GravityContractSeed)
	if err != nil {
		return nil, err
	}

	testConfig.Nebula, err = GenerateAddressFromSeed(cfg.ChainId, cfg.NebulaContractSeed)
	if err != nil {
		return nil, err
	}

	testConfig.Sub, err = GenerateAddressFromSeed(cfg.ChainId, cfg.SubscriberContractSeed)
	if err != nil {
		return nil, err
	}

	gravityScript, err := ScriptFromFile(cfg.GravityScriptFile)
	if err != nil {
		return nil, err
	}

	nebulaScript, err := ScriptFromFile(cfg.NebulaScriptFile)
	if err != nil {
		return nil, err
	}

	subScript, err := ScriptFromFile(cfg.SubMockScriptFile)
	if err != nil {
		return nil, err
	}

	wCrypto := wavesCrypto.NewWavesCrypto()
	distributionSeed, err := crypto.NewSecretKeyFromBase58(string(wCrypto.PrivateKey(wavesCrypto.Seed(cfg.DistributionSeed))))
	if err != nil {
		return nil, err
	}

	gravityAddressRecipient, err := proto.NewRecipientFromString(testConfig.Gravity.Address)
	if err != nil {
		return nil, err
	}
	nebulaAddressRecipient, err := proto.NewRecipientFromString(testConfig.Nebula.Address)
	if err != nil {
		return nil, err
	}
	subAddressRecipient, err := proto.NewRecipientFromString(testConfig.Sub.Address)
	if err != nil {
		return nil, err
	}

	massTx := &proto.MassTransferWithProofs{
		Type:      proto.MassTransferTransaction,
		Version:   1,
		SenderPK:  crypto.GeneratePublicKey(distributionSeed),
		Fee:       5000000,
		Timestamp: wavesClient.NewTimestampFromTime(time.Now()),
		Transfers: []proto.MassTransferEntry{
			{
				Amount:    2 * Wavelet,
				Recipient: gravityAddressRecipient,
			},
			{
				Amount:    2 * Wavelet,
				Recipient: nebulaAddressRecipient,
			},
			{
				Amount:    2 * Wavelet,
				Recipient: subAddressRecipient,
			},
		},
		Attachment: &proto.LegacyAttachment{},
	}
	err = massTx.Sign(cfg.ChainId, distributionSeed)
	if err != nil {
		return nil, err
	}
	_, err = testConfig.Client.Transactions.Broadcast(testConfig.Ctx, massTx)
	if err != nil {
		return nil, err
	}
	err = <-testConfig.Helper.WaitTx(massTx.ID.String(), testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	var consulsString []string
	for _, v := range testConfig.Consuls {
		consulsString = append(consulsString, v)
	}
	err = deployer.DeployGravityWaves(testConfig.Client, testConfig.Helper, gravityScript, consulsString, BftValue, cfg.ChainId, testConfig.Gravity.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	err = deployer.DeploySubWaves(testConfig.Client, testConfig.Helper, subScript, cfg.ChainId, testConfig.Sub.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	oraclesString := make([]string, 0)
	err = deployer.DeployNebulaWaves(testConfig.Client, testConfig.Helper, nebulaScript, testConfig.Gravity.Address,
		testConfig.Sub.Address, oraclesString, BftValue, contracts.BytesType, cfg.ChainId, testConfig.Nebula.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	return &testConfig, nil
}