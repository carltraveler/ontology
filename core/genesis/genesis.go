/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package genesis

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/constants"
	"github.com/ontio/ontology/consensus/vbft/config"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/core/utils"
	"github.com/ontio/ontology/smartcontract/service/native/global_params"
	"github.com/ontio/ontology/smartcontract/service/native/governance"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	nutils "github.com/ontio/ontology/smartcontract/service/native/utils"
	"github.com/ontio/ontology/smartcontract/service/neovm"
)

const (
	BlockVersion uint32 = 0
	GenesisNonce uint64 = 2083236893
)

var (
	ONTToken   = newGoverningToken()
	ONGToken   = newUtilityToken()
	ONTTokenID = ONTToken.Hash()
	ONGTokenID = ONGToken.Hash()
)

var GenBlockTime = (config.DEFAULT_GEN_BLOCK_TIME * time.Second)

var INIT_PARAM = map[string]string{
	"gasPrice": "0",
}

var GenesisBookkeepers []keypair.PublicKey

// BuildGenesisBlock returns the genesis block with default consensus bookkeeper list
func BuildGenesisBlock(defaultBookkeeper []keypair.PublicKey, genesisConfig *config.GenesisConfig) (*types.Block, error) {
	//getBookkeeper
	GenesisBookkeepers = defaultBookkeeper
	nextBookkeeper, err := types.AddressFromBookkeepers(defaultBookkeeper)
	if err != nil {
		return nil, fmt.Errorf("[Block],BuildGenesisBlock err with GetBookkeeperAddress: %s", err)
	}
	conf := bytes.NewBuffer(nil)
	if genesisConfig.VBFT != nil {
		//序列化vbft config.供治理合约初始化使用
		genesisConfig.VBFT.Serialize(conf)
	}
	//传入vbft的全局配置，如共识节点的地址，写入的共识合约的账本空间，由于所有的落账都需要通过tx事务来运行，所以这里要构造初始化治理合约账本空间，将vbft写入，需要构造tx。疑问是，gas由谁来付
	//构造创世区块包含的交易.
	govConfig := newGoverConfigInit(conf.Bytes())
	//consensusPayload是VbftBlockInfo的json序列化数据
	consensusPayload, err := vconfig.GenesisConsensusPayload(govConfig.Hash(), 0) //第三个参数是blockheight，可知，第一个tx所在的block永远是初始化治理合约账本空间.并在该账本空间中存入vbft config序列化数据.
	if err != nil {
		return nil, fmt.Errorf("consensus genesus init failed: %s", err)
	}
	//blockdata, 构造创始区块头部
	genesisHeader := &types.Header{
		Version:          BlockVersion,
		PrevBlockHash:    common.Uint256{},
		TransactionsRoot: common.Uint256{},
		Timestamp:        constants.GENESIS_BLOCK_TIMESTAMP,
		Height:           uint32(0), //创世区块的高度永远是0
		ConsensusData:    GenesisNonce,
		NextBookkeeper:   nextBookkeeper,
		ConsensusPayload: consensusPayload,

		Bookkeepers: nil,
		SigData:     nil,
	}

	//block. 创始区块交易
	ont := newGoverningToken() //只是将ont native合约的地址作为paload code 部署到链上去.打包成deploy Transaction
	ong := newUtilityToken()
	param := newParamContract()
	oid := deployOntIDContract()
	auth := deployAuthContract()
	config := newConfig() //治理合约地址

	genesisBlock := &types.Block{
		Header: genesisHeader, //创始区块头部
		Transactions: []*types.Transaction{ //创始区块包含的交易。
			ont,
			ong,
			param,
			oid,
			auth,
			config,
			newGoverningInit(), //ont init tx. 参数是Bookkeeper多签地址，和ont总发行量
			newUtilityInit(),   //ong init tx. 无参数
			newParamInit(),     //param init tx， 参数:Bookkeeper多签地址
			govConfig,          //构造调用治理合约init方法的事务交易,将vbft共识配置，共识节点地址序列化的数据落入治理合约账本空间，上链
		},
	}
	genesisBlock.RebuildMerkleRoot()
	return genesisBlock, nil
}

func newGoverningToken() *types.Transaction {
	mutable := utils.NewDeployTransaction(nutils.OntContractAddress[:], "ONT", "1.0",
		"Ontology Team", "contact@ont.io", "Ontology Network ONT Token", true)
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis governing token transaction error ")
	}
	return tx
}

func newUtilityToken() *types.Transaction {
	mutable := utils.NewDeployTransaction(nutils.OngContractAddress[:], "ONG", "1.0",
		"Ontology Team", "contact@ont.io", "Ontology Network ONG Token", true)
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis utility token transaction error ")
	}
	return tx
}

func newParamContract() *types.Transaction {
	mutable := utils.NewDeployTransaction(nutils.ParamContractAddress[:],
		"ParamConfig", "1.0", "Ontology Team", "contact@ont.io",
		"Chain Global Environment Variables Manager ", true)
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis param transaction error ")
	}
	return tx
}

func newConfig() *types.Transaction {
	mutable := utils.NewDeployTransaction(nutils.GovernanceContractAddress[:], "CONFIG", "1.0",
		"Ontology Team", "contact@ont.io", "Ontology Network Consensus Config", true)
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis config transaction error ")
	}
	return tx
}

func deployAuthContract() *types.Transaction {
	mutable := utils.NewDeployTransaction(nutils.AuthContractAddress[:], "AuthContract", "1.0",
		"Ontology Team", "contact@ont.io", "Ontology Network Authorization Contract", true)
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis auth transaction error ")
	}
	return tx
}

func deployOntIDContract() *types.Transaction {
	mutable := utils.NewDeployTransaction(nutils.OntIDContractAddress[:], "OID", "1.0",
		"Ontology Team", "contact@ont.io", "Ontology Network ONT ID", true)
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis ontid transaction error ")
	}
	return tx
}

func newGoverningInit() *types.Transaction {
	bookkeepers, _ := config.DefConfig.GetBookkeepers()

	var addr common.Address
	if len(bookkeepers) == 1 {
		addr = types.AddressFromPubKey(bookkeepers[0])
	} else {
		m := (5*len(bookkeepers) + 6) / 7
		temp, err := types.AddressFromMultiPubKeys(bookkeepers, m)
		if err != nil {
			panic(fmt.Sprint("wrong bookkeeper config, caused by", err))
		}
		addr = temp
	}

	distribute := []struct {
		addr  common.Address
		value uint64
	}{{addr, constants.ONT_TOTAL_SUPPLY}}

	args := bytes.NewBuffer(nil)
	nutils.WriteVarUint(args, uint64(len(distribute)))
	for _, part := range distribute {
		nutils.WriteAddress(args, part.addr)
		nutils.WriteVarUint(args, part.value)
	}

	mutable := utils.BuildNativeTransaction(nutils.OntContractAddress, ont.INIT_NAME, args.Bytes())
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis governing token transaction error ")
	}
	return tx
}

func newUtilityInit() *types.Transaction {
	mutable := utils.BuildNativeTransaction(nutils.OngContractAddress, ont.INIT_NAME, []byte{})
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis governing token transaction error ")
	}

	return tx
}

func newParamInit() *types.Transaction {
	params := new(global_params.Params)
	var s []string
	for k := range INIT_PARAM {
		s = append(s, k)
	}

	for k, v := range neovm.INIT_GAS_TABLE {
		INIT_PARAM[k] = strconv.FormatUint(v, 10)
		s = append(s, k)
	}

	sort.Strings(s)
	for _, v := range s {
		params.SetParam(global_params.Param{Key: v, Value: INIT_PARAM[v]})
	}
	bf := new(bytes.Buffer)
	params.Serialize(bf)

	bookkeepers, _ := config.DefConfig.GetBookkeepers()
	var addr common.Address
	if len(bookkeepers) == 1 {
		addr = types.AddressFromPubKey(bookkeepers[0])
	} else {
		m := (5*len(bookkeepers) + 6) / 7
		temp, err := types.AddressFromMultiPubKeys(bookkeepers, m)
		if err != nil {
			panic(fmt.Sprint("wrong bookkeeper config, caused by", err))
		}
		addr = temp
	}
	nutils.WriteAddress(bf, addr)

	mutable := utils.BuildNativeTransaction(nutils.ParamContractAddress, global_params.INIT_NAME, bf.Bytes())
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis governing token transaction error ")
	}
	return tx
}

//由于所有的存储
func newGoverConfigInit(config []byte) *types.Transaction {
	mutable := utils.BuildNativeTransaction(nutils.GovernanceContractAddress, governance.INIT_CONFIG, config)
	tx, err := mutable.IntoImmutable()
	if err != nil {
		panic("constract genesis governing token transaction error ")
	}
	return tx
}
