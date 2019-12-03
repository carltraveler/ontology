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
package wasmvm

import (
	"sync"
	"unsafe"

	"github.com/go-interpreter/wagon/exec"
	"github.com/hashicorp/golang-lru"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/store"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/context"
	"github.com/ontio/ontology/smartcontract/event"
	"github.com/ontio/ontology/smartcontract/states"
	"github.com/ontio/ontology/smartcontract/storage"
)

type WasmVmService struct {
	Store         store.LedgerStore
	CacheDB       *storage.CacheDB
	ContextRef    context.ContextRef
	Notifications []*event.NotifyEventInfo
	Code          []byte
	Tx            *types.Transaction
	Time          uint32
	Height        uint32
	BlockHash     common.Uint256
	PreExec       bool
	GasPrice      uint64
	GasLimit      *uint64
	ExecStep      *uint64
	GasFactor     uint64
	IsTerminate   bool
	ServiceIndex  uint64
	vm            *exec.VM
}

var (
	ERR_CHECK_STACK_SIZE  = errors.NewErr("[WasmVmService] vm over max stack size!")
	ERR_EXECUTE_CODE      = errors.NewErr("[WasmVmService] vm execute code invalid!")
	ERR_GAS_INSUFFICIENT  = errors.NewErr("[WasmVmService] gas insufficient")
	VM_EXEC_STEP_EXCEED   = errors.NewErr("[WasmVmService] vm execute step exceed!")
	CONTRACT_NOT_EXIST    = errors.NewErr("[WasmVmService] Get contract code from db fail")
	DEPLOYCODE_TYPE_ERROR = errors.NewErr("[WasmVmService] DeployCode type error!")
	VM_EXEC_FAULT         = errors.NewErr("[WasmVmService] vm execute state fault!")
	VM_INIT_FAULT         = errors.NewErr("[WasmVmService] vm init state fault!")

	CODE_CACHE_SIZE      = 100
	CONTRACT_METHOD_NAME = "invoke"

	//max memory size of wasm vm
	WASM_MEM_LIMITATION  uint64 = 10 * 1024 * 1024
	VM_STEP_LIMIT               = 40000000
	WASM_CALLSTACK_LIMIT        = 1024

	CodeCache *lru.ARCCache

	ctxData        = make(map[uint64]*WasmVmService)
	nextCtxDataIdx uint64
	ctxDataMtx     sync.RWMutex
)

func init() {
	CodeCache, _ = lru.NewARC(CODE_CACHE_SIZE)
	nextCtxDataIdx = 0
	//if err != nil{
	//	log.Info("NewARC block error %s", err)
	//}
}

func GetAddressBuff(addrs []common.Address) ([]byte, int) {
	addrsLen := len(addrs) * 20
	addrsBuff := make([]byte, addrsLen)
	for off, addr := range addrs {
		ptr := (*[20]byte)(unsafe.Pointer(&addr[0]))
		copy(addrsBuff[off*20:off*20+20], (*ptr)[:])
	}

	return addrsBuff, addrsLen
}

func (this *WasmVmService) SetContextData() {
	defer ctxDataMtx.Unlock()
	ctxDataMtx.Lock()
	ctxData[nextCtxDataIdx] = this
	this.ServiceIndex = nextCtxDataIdx
	nextCtxDataIdx++
}

func GetWasmVmService(index uint64) *WasmVmService {
	defer ctxDataMtx.Unlock()
	ctxDataMtx.Lock()
	return ctxData[index]
}

func (this *WasmVmService) Invoke() (interface{}, error) {
	if len(this.Code) == 0 {
		return nil, ERR_EXECUTE_CODE
	}

	contract := &states.WasmContractParam{}
	sink := common.NewZeroCopySource(this.Code)
	err := contract.Deserialization(sink)
	if err != nil {
		return nil, err
	}

	code, err := this.CacheDB.GetContract(contract.Address)
	if err != nil {
		return nil, err
	}

	if code == nil {
		return nil, errors.NewErr("wasm contract does not exist")
	}

	wasmCode, err := code.GetWasmCode()
	if err != nil {
		return nil, errors.NewErr("not a wasm contract")
	}

	this.ContextRef.PushContext(&context.Context{ContractAddress: contract.Address, Code: wasmCode})

	output, err := invokeJit(this, contract, wasmCode)
	if err != nil {
		return nil, err
	}

	this.ContextRef.PopContext()
	return output, nil
}
