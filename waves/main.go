package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	wavesCrypto "github.com/wavesplatform/go-lib-crypto"

	"github.com/Gravity-Tech/gateway-deployer/waves/contracts"
	"github.com/Gravity-Tech/gateway-deployer/waves/deployer"

	"github.com/wavesplatform/gowaves/pkg/proto"

	"github.com/wavesplatform/gowaves/pkg/crypto"

	"github.com/Gravity-Tech/gravity-core/common/helpers"
	wavesClient "github.com/wavesplatform/gowaves/pkg/client"
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
		Wavelet = 1e8
	)

	var testConfig DeploymentConfig
	testConfig.Ctx = context.Background()

	cfg, err := LoadDeploymentConfig(configFile)
	if err != nil {
		return nil, err
	}

	if cfg.AssetID == "" {
		return nil, fmt.Errorf("valid asset id is not provided")
	}

	wClient, err := wavesClient.NewClient(wavesClient.Options{ApiKey: "", BaseUrl: cfg.NodeUrl})
	if err != nil {
		return nil, err
	}
	testConfig.Client = wClient
	testConfig.Helper = helpers.NewClientHelper(testConfig.Client)

	testConfig.Consuls = make([]string, 5, 5)
	for i := 0; i < 5; i++ {
		if i < len(cfg.ConsulsPubKeys) {
			testConfig.Consuls[i] = cfg.ConsulsPubKeys[i]
		} else {
			testConfig.Consuls[i] = "1"
		}
	}

	testConfig.Nebula, err = GenerateAddressFromSeed(cfg.ChainId, cfg.NebulaContractSeed)
	if err != nil {
		return nil, err
	}

	testConfig.Sub, err = GenerateAddressFromSeed(cfg.ChainId, cfg.SubscriberContractSeed)
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
		Fee:       0.005 * Wavelet,
		Timestamp: wavesClient.NewTimestampFromTime(time.Now()),
		Transfers: []proto.MassTransferEntry{
			{
				Amount:    0.5 * Wavelet,
				Recipient: nebulaAddressRecipient,
			},
			{
				Amount:    0.5 * Wavelet,
				Recipient: subAddressRecipient,
			},
		},
		Attachment: proto.Attachment{},
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

	err = deployer.DeploySubWaves(testConfig.Client, testConfig.Helper, subScript, nebulaAddressRecipient.String(), cfg.AssetID, cfg.ChainId, testConfig.Sub.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	oraclesString := consulsString[:]
	err = deployer.DeployNebulaWaves(testConfig.Client, testConfig.Helper, nebulaScript, cfg.ExistingGravityAddress,
		testConfig.Sub.Address, oraclesString, cfg.BftValue, contracts.BytesType, cfg.ChainId, testConfig.Nebula.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	return &testConfig, nil
}
