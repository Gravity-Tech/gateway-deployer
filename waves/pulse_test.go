package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/Gravity-Tech/gateway-deployer/waves/helper"
	"math/rand"
	"strings"
	"testing"
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
	BftValue    = 3
	ConsulCount = 5
	OracleCount = 5
	Wavelet     = 100000000
)

var config *helper.TestPulseConfig
var tests = map[string]func(t *testing.T){
	"sendHashPositive":     testSendHashPositive,
	"sendHashInvalidSigns": testSendHashInvalidSigns,
	"sendSubPositive":      testSendSubPositive,
	"sendSubInvalidHash":   testSendSubInvalidHash,
}

func TestPulse(t *testing.T) {
	var err error
	config, err = initTests()
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range tests {
		height, _, err := config.Client.Blocks.Height(config.Ctx)
		if err != nil {
			t.Fatal(err)
		}

		t.Run(k, v)

		err = <-config.Helper.WaitByHeight(height.Height+1, config.Ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func initTests() (*helper.TestPulseConfig, error) {
	var testConfig helper.TestPulseConfig
	testConfig.Ctx = context.Background()

	cfg, err := helper.LoadConfig("config.json")
	if err != nil {
		return nil, err
	}

	wClient, err := wavesClient.NewClient(wavesClient.Options{ApiKey: "", BaseUrl: cfg.NodeUrl})
	if err != nil {
		return nil, err
	}
	testConfig.Client = wClient
	testConfig.Helper = helpers.NewClientHelper(testConfig.Client)

	testConfig.Gravity, err = helper.GenerateAddress(cfg.ChainId)
	if err != nil {
		return nil, err
	}

	testConfig.Nebula, err = helper.GenerateAddress(cfg.ChainId)
	if err != nil {
		return nil, err
	}

	testConfig.Sub, err = helper.GenerateAddress(cfg.ChainId)
	if err != nil {
		return nil, err
	}

	for i := 0; i < ConsulCount; i++ {
		consul, err := helper.GenerateAddress(cfg.ChainId)
		if err != nil {
			return nil, err
		}

		testConfig.Consuls = append(testConfig.Consuls, consul)
	}
	for i := 0; i < OracleCount; i++ {
		oracle, err := helper.GenerateAddress(cfg.ChainId)
		if err != nil {
			return nil, err
		}

		testConfig.Oracles = append(testConfig.Consuls, oracle)
	}

	gravityScript, err := helper.ScriptFromFile(cfg.GravityScriptFile)
	if err != nil {
		return nil, err
	}

	nebulaScript, err := helper.ScriptFromFile(cfg.NebulaScriptFile)
	if err != nil {
		return nil, err
	}

	subScript, err := helper.ScriptFromFile(cfg.SubMockScriptFile)
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
	oracleRecipient, err := proto.NewRecipientFromString(testConfig.Oracles[0].Address)
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
			{
				Amount:    2 * Wavelet,
				Recipient: oracleRecipient,
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
		consulsString = append(consulsString, v.Address)
	}
	err = deployer.DeployGravityWaves(testConfig.Client, testConfig.Helper, gravityScript, consulsString, BftValue, cfg.ChainId, testConfig.Gravity.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	err = deployer.DeploySubWaves(testConfig.Client, testConfig.Helper, subScript, "Nebula", "assetId", cfg.ChainId, testConfig.Sub.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	var oraclesString []string
	for _, v := range testConfig.Oracles {
		oraclesString = append(oraclesString, v.PubKey.String())
	}
	err = deployer.DeployNebulaWaves(testConfig.Client, testConfig.Helper, nebulaScript, testConfig.Gravity.Address,
		testConfig.Sub.Address, oraclesString, BftValue, contracts.BytesType, cfg.ChainId, testConfig.Nebula.Secret, testConfig.Ctx)
	if err != nil {
		return nil, err
	}

	return &testConfig, nil
}

func testSendHashPositive(t *testing.T) {
	cfg, _ := helper.LoadConfig("config.json")

	id := make([]byte, 32)
	_, err := rand.Read(id)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.Keccak256(id)
	if err != nil {
		t.Fatal(err)
	}

	var signs []string
	for _, v := range config.Oracles {
		sign, err := crypto.Sign(v.Secret, hash.Bytes())
		if err != nil {
			t.Fatal(err)
		}

		signs = append(signs, sign.String())
	}

	recipient, err := proto.NewAddressFromString(config.Nebula.Address)
	if err != nil {
		t.Fatal(err)
	}

	tx := &proto.InvokeScriptWithProofs{
		Type:     proto.InvokeScriptTransaction,
		Version:  1,
		SenderPK: config.Oracles[0].PubKey,
		ChainID:  cfg.ChainId,
		FunctionCall: proto.FunctionCall{
			Name: "sendHashValue",
			Arguments: proto.Arguments{
				proto.BinaryArgument{
					Value: hash.Bytes(),
				},
				proto.StringArgument{
					Value: strings.Join(signs, ","),
				},
			},
		},
		Fee:             5000000,
		Timestamp:       wavesClient.NewTimestampFromTime(time.Now()),
		ScriptRecipient: proto.NewRecipientFromAddress(recipient),
	}
	err = tx.Sign(cfg.ChainId, config.Oracles[0].Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = config.Client.Transactions.Broadcast(config.Ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	err = <-config.Helper.WaitTx(tx.ID.String(), config.Ctx)
	if err != nil {
		t.Fatal(err)
	}
}
func testSendHashInvalidSigns(t *testing.T) {
	cfg, _ := helper.LoadConfig("config.json")

	id := make([]byte, 32)
	_, err := rand.Read(id)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.Keccak256(id)
	if err != nil {
		t.Fatal(err)
	}

	var signs []string
	for i := 0; i < OracleCount; i++ {
		if i >= (BftValue - 1) {
			signs = append(signs, "")
			continue
		}
		sign, err := crypto.Sign(config.Oracles[i].Secret, hash.Bytes())
		if err != nil {
			t.Fatal(err)
		}

		signs = append(signs, sign.String())
	}

	recipient, err := proto.NewAddressFromString(config.Nebula.Address)
	if err != nil {
		t.Fatal(err)
	}

	tx := &proto.InvokeScriptWithProofs{
		Type:     proto.InvokeScriptTransaction,
		Version:  1,
		SenderPK: config.Oracles[0].PubKey,
		ChainID:  cfg.ChainId,
		FunctionCall: proto.FunctionCall{
			Name: "sendHashValue",
			Arguments: proto.Arguments{
				proto.BinaryArgument{
					Value: hash.Bytes(),
				},
				proto.StringArgument{
					Value: strings.Join(signs, ","),
				},
			},
		},
		Fee:             5000000,
		Timestamp:       wavesClient.NewTimestampFromTime(time.Now()),
		ScriptRecipient: proto.NewRecipientFromAddress(recipient),
	}
	err = tx.Sign(cfg.ChainId, config.Oracles[0].Secret)
	if err != nil {
		t.Fatal(err)
	}

	_, err = config.Client.Transactions.Broadcast(config.Ctx, tx)
	if err == nil {
		t.Fatal("invalid signs not fail in contract")
	}

	err = helper.CheckRideError(err, "invalid bft count")
	if err != nil {
		t.Fatal(err)
	}
}

func testSendSubPositive(t *testing.T) {
	cfg, _ := helper.LoadConfig("config.json")

	id := make([]byte, 32)
	_, err := rand.Read(id)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.Keccak256(id)
	if err != nil {
		t.Fatal(err)
	}

	var signs []string
	for _, v := range config.Oracles {
		sign, err := crypto.Sign(v.Secret, hash.Bytes())
		if err != nil {
			t.Fatal(err)
		}

		signs = append(signs, sign.String())
	}

	recipient, err := proto.NewAddressFromString(config.Nebula.Address)
	if err != nil {
		t.Fatal(err)
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:     proto.InvokeScriptTransaction,
		Version:  1,
		SenderPK: config.Oracles[0].PubKey,
		ChainID:  cfg.ChainId,
		FunctionCall: proto.FunctionCall{
			Name: "sendHashValue",
			Arguments: proto.Arguments{
				proto.BinaryArgument{
					Value: hash.Bytes(),
				},
				proto.StringArgument{
					Value: strings.Join(signs, ","),
				},
			},
		},
		Fee:             5000000,
		Timestamp:       wavesClient.NewTimestampFromTime(time.Now()),
		ScriptRecipient: proto.NewRecipientFromAddress(recipient),
	}

	err = tx.Sign(cfg.ChainId, config.Oracles[0].Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = config.Client.Transactions.Broadcast(config.Ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	err = <-config.Helper.WaitTx(tx.ID.String(), config.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	lastPulseState, _, err := config.Helper.GetStateByAddressAndKey(config.Nebula.Address, "last_pulse_id", config.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	lastPulseId := int64(lastPulseState.Value.(float64))
	recipient, err = proto.NewAddressFromString(config.Sub.Address)
	if err != nil {
		t.Fatal(err)
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:     proto.InvokeScriptTransaction,
		Version:  1,
		SenderPK: config.Nebula.PubKey,
		ChainID:  cfg.ChainId,
		FunctionCall: proto.FunctionCall{
			Name: "attachValue",
			Arguments: proto.Arguments{
				proto.BinaryArgument{
					Value: id,
				},
				proto.IntegerArgument{
					Value: lastPulseId,
				},
			},
		},
		Fee:             5000000,
		Timestamp:       wavesClient.NewTimestampFromTime(time.Now()),
		ScriptRecipient: proto.NewRecipientFromAddress(recipient),
	}

	err = tx.Sign(cfg.ChainId, config.Oracles[0].Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = config.Client.Transactions.Broadcast(config.Ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	err = <-config.Helper.WaitTx(tx.ID.String(), config.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	subValue, _, err := config.Helper.GetStateByAddressAndKey(config.Sub.Address, fmt.Sprintf("%d", lastPulseId), config.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	value, err := base64.StdEncoding.DecodeString(strings.Split(subValue.Value.(string), ":")[1])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(value, id) {
		t.Fatal("invalid sent value")
	}
}
func testSendSubInvalidHash(t *testing.T) {
	cfg, _ := helper.LoadConfig("config.json")

	id := make([]byte, 32)
	_, err := rand.Read(id)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.Keccak256(id)
	if err != nil {
		t.Fatal(err)
	}

	var signs []string
	for _, v := range config.Oracles {
		sign, err := crypto.Sign(v.Secret, hash.Bytes())
		if err != nil {
			t.Fatal(err)
		}

		signs = append(signs, sign.String())
	}

	recipient, err := proto.NewAddressFromString(config.Nebula.Address)
	if err != nil {
		t.Fatal(err)
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:     proto.InvokeScriptTransaction,
		Version:  1,
		SenderPK: config.Oracles[0].PubKey,
		ChainID:  cfg.ChainId,
		FunctionCall: proto.FunctionCall{
			Name: "sendHashValue",
			Arguments: proto.Arguments{
				proto.BinaryArgument{
					Value: hash.Bytes(),
				},
				proto.StringArgument{
					Value: strings.Join(signs, ","),
				},
			},
		},
		Fee:             5000000,
		Timestamp:       wavesClient.NewTimestampFromTime(time.Now()),
		ScriptRecipient: proto.NewRecipientFromAddress(recipient),
	}

	err = tx.Sign(cfg.ChainId, config.Oracles[0].Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = config.Client.Transactions.Broadcast(config.Ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	err = <-config.Helper.WaitTx(tx.ID.String(), config.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	lastPulseId, _, err := config.Helper.GetStateByAddressAndKey(config.Nebula.Address, "last_pulse_id", config.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	lastPulse := int64(lastPulseId.Value.(float64))
	recipient, err = proto.NewAddressFromString(config.Sub.Address)
	if err != nil {
		t.Fatal(err)
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:     proto.InvokeScriptTransaction,
		Version:  1,
		SenderPK: config.Nebula.PubKey,
		ChainID:  cfg.ChainId,
		FunctionCall: proto.FunctionCall{
			Name: "attachValue",
			Arguments: proto.Arguments{
				proto.BinaryArgument{
					Value: make([]byte, 32, 32),
				},
				proto.IntegerArgument{
					Value: lastPulse,
				},
			},
		},
		Fee:             5000000,
		Timestamp:       wavesClient.NewTimestampFromTime(time.Now()),
		ScriptRecipient: proto.NewRecipientFromAddress(recipient),
	}

	err = tx.Sign(cfg.ChainId, config.Oracles[0].Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = config.Client.Transactions.Broadcast(config.Ctx, tx)
	if err == nil {
		t.Fatal("invalid value is sent")
	}
	err = helper.CheckRideError(err, "invalid keccak256(value)")
	if err != nil {
		t.Fatal(err)
	}
}
