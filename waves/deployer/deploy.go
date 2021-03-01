package deployer

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Gravity-Tech/gateway-deployer/waves/contracts"

	wavesHelper "github.com/Gravity-Tech/gravity-core/common/helpers"

	wavesClient "github.com/wavesplatform/gowaves/pkg/client"
	wavesCrypto "github.com/wavesplatform/gowaves/pkg/crypto"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func DeployGravityWaves(
	client *wavesClient.Client,
	helper wavesHelper.ClientHelper,
	gravityScript []byte,
	consulsPubKeys []string,
	bftValue int64,
	chainId byte,
	secret wavesCrypto.SecretKey,
	ctx context.Context,
) error {
	id, err := DeployWavesContract(client, gravityScript, chainId, secret, ctx)

	if err != nil {
		return err
	}

	err = <-helper.WaitTx(id, ctx)
	if err != nil {
		return err
	}

	log.Println("gravity tx", id)

	id, err = DataWavesContract(client, chainId, secret, proto.DataEntries{
		&proto.StringDataEntry{
			Key:   "consuls_0",
			Value: strings.Join(consulsPubKeys, ","),
		},
		&proto.IntegerDataEntry{
			Key:   "last_round",
			Value: 0,
		},
		&proto.IntegerDataEntry{
			Key:   "bft_coefficient",
			Value: bftValue,
		},
	}, ctx)
	if err != nil {
		return err
	}

	err = <-helper.WaitTx(id, ctx)
	if err != nil {
		return err
	}

	return nil
}

func DeployNebulaWaves(client *wavesClient.Client, helper wavesHelper.ClientHelper, nebulaScript []byte, gravityAddress string, subscriberAddress string,
	oracles []string, bftValue int64, dataType contracts.ExtractorType, chainId byte, secret wavesCrypto.SecretKey, ctx context.Context) error {

	id, err := DeployWavesContract(client, nebulaScript, chainId, secret, ctx)
	if err != nil {
		return err
	}

	err = <-helper.WaitTx(id, ctx)
	if err != nil {
		return err
	}

	log.Println("nebula tx", id)

	id, err = DataWavesContract(client, chainId, secret, proto.DataEntries{
		&proto.StringDataEntry{
			Key:   "oracles",
			Value: strings.Join(oracles, ","),
		},
		&proto.IntegerDataEntry{
			Key:   "bft_coefficient",
			Value: bftValue,
		},
		&proto.StringDataEntry{
            Key:   "contract_pubkey",
            Value: wavesCrypto.GeneratePublicKey(secret).String(),
        },
		&proto.IntegerDataEntry{
            Key:   "last_round",
            Value: 0,
        },
		&proto.StringDataEntry{
			Key:   "subscriber_address",
			Value: subscriberAddress,
		},
		&proto.StringDataEntry{
			Key:   "gravity_contract",
			Value: gravityAddress,
		},
		&proto.IntegerDataEntry{
			Key:   "type",
			Value: int64(dataType),
		},
	}, ctx)
	if err != nil {
		return err
	}

	err = <-helper.WaitTx(id, ctx)
	if err != nil {
		return err
	}

	return nil
}
// Subscriber
func DeploySubWaves(
	client *wavesClient.Client,
	helper wavesHelper.ClientHelper,
	subScript []byte,
	nebulaAddress string,
	assetId string,
	chainId byte,
	secret wavesCrypto.SecretKey,
	ctx context.Context,
) error {
	id, err := DeployWavesContract(client, subScript, chainId, secret, ctx)
	if err != nil {
		return err
	}

	// Script deployment
	err = <-helper.WaitTx(id, ctx)
	if err != nil {
		return err
	}

	id, err = DataWavesContract(client, chainId, secret, proto.DataEntries{
		&proto.StringDataEntry{
            Key:   "nebula_address",
            Value: nebulaAddress,
        },
        &proto.StringDataEntry{
            Key:   "asset_id",
            Value: assetId,
        },
        &proto.IntegerDataEntry{
            Key:   "type",
            Value: 2, // byte type
        },
    }, ctx)

    if err != nil {
        return err
    }

    err = <-helper.WaitTx(id, ctx)
    if err != nil {
        return err
    }

	return nil
}

func DeployWavesContract(client *wavesClient.Client, contactScript []byte, chainId byte, secret wavesCrypto.SecretKey, ctx context.Context) (string, error) {
	tx := &proto.SetScriptWithProofs{
		Type:      proto.SetScriptTransaction,
		Version:   1,
		SenderPK:  wavesCrypto.GeneratePublicKey(secret),
		ChainID:   chainId,
		Script:    contactScript,
		Fee:       10000000,
		Timestamp: wavesClient.NewTimestampFromTime(time.Now()),
	}
	err := tx.Sign(chainId, secret)
	if err != nil {
		return "", err
	}

	_, err = client.Transactions.Broadcast(ctx, tx)
	if err != nil {
		return "", err
	}

	return tx.ID.String(), nil
}

func DataWavesContract(client *wavesClient.Client, chainId byte, secret wavesCrypto.SecretKey, dataEntries proto.DataEntries, ctx context.Context) (string, error) {
	tx := &proto.DataWithProofs{
		Type:      proto.DataTransaction,
		Version:   1,
		SenderPK:  wavesCrypto.GeneratePublicKey(secret),
		Entries:   dataEntries,
		Fee:       10000000,
		Timestamp: wavesClient.NewTimestampFromTime(time.Now()),
	}

	err := tx.Sign(chainId, secret)
	if err != nil {
		return "", err
	}

	_, err = client.Transactions.Broadcast(ctx, tx)
	if err != nil {
		return "", err
	}

	return tx.ID.String(), nil
}
