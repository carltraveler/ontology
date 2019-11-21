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

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L. -lwasmjit -ldl
#include "wasm_service.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/go-interpreter/wagon/exec"
	"github.com/hashicorp/golang-lru"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/store"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/context"
	"github.com/ontio/ontology/smartcontract/event"
	"github.com/ontio/ontology/smartcontract/states"
	"github.com/ontio/ontology/smartcontract/storage"
)

type WasmVmService struct {
	Store            store.LedgerStore
	CacheDB          *storage.CacheDB
	ContextRef       context.ContextRef
	Notifications    []*event.NotifyEventInfo
	Code             []byte
	Tx               *types.Transaction
	Time             uint32
	Height           uint32
	BlockHash        common.Uint256
	PreExec          bool
	GasPrice         uint64
	GasLimit         *uint64
	ExecStep         *uint64
	GasFactor        uint64
	IsTerminate      bool
	wasmVmServicePtr uint64
	vm               *exec.VM
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

//export ontio_debug_cgo
func ontio_debug_cgo(vmctx *C.uchar, data_ptr uint32, data_len uint32) {
	fmt.Printf("DebugCgo enter\n")
	bs := make([]byte, data_len)
	C.ontio_read_wasmvm_memory(vmctx, (*C.uchar)(unsafe.Pointer(&bs[0])), C.uint(data_ptr), C.uint(data_len))
	log.Debugf("[WasmContract]Debug:%s\n", bs)
}

func (this *WasmVmService) SetContextData() {
	ctxDataMtx.Lock()
	ctxData[nextCtxDataIdx] = this
	this.wasmVmServicePtr = nextCtxDataIdx
	nextCtxDataIdx++
	defer ctxDataMtx.Unlock()
}

func (this *WasmVmService) Invoke() (interface{}, error) {
	fmt.Printf("wasm invoke enter\n")
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

	fmt.Printf("blockhash :%v\n", this.BlockHash)

	txHash := this.Tx.Hash()
	witnessAddrBuff, witnessAddrBuffLen := GetAddressBuff(this.Tx.GetSignatureAddresses())
	callersAddrBuff, callersAddrBuffLen := GetAddressBuff(this.ContextRef.GetCallerAddress())
	fmt.Printf("witnessAddrBuffLen: %d\n", witnessAddrBuffLen/20)
	fmt.Printf("callersAddrLen : %d\n", callersAddrBuffLen/20)

	var witnessptr *C.uchar

	if witnessAddrBuffLen == 0 {
		witnessptr = (*C.uchar)((unsafe.Pointer)(nil))
	} else {
		witnessptr = (*C.uchar)((unsafe.Pointer)(&witnessAddrBuff[0]))
	}

	inter_chain := C.InterOpCtx{
		height:             C.uint(this.Height),
		block_hash:         (*C.uchar)((unsafe.Pointer)(&this.BlockHash[0])),
		timestamp:          C.ulonglong(this.Time),
		tx_hash:            (*C.uchar)((unsafe.Pointer)(&(txHash[0]))),
		self_address:       (*C.uchar)((unsafe.Pointer)(&contract.Address[0])),
		callers:            (*C.uchar)((unsafe.Pointer)(&callersAddrBuff[0])),
		callers_num:        C.ulong(callersAddrBuffLen),
		witness:            witnessptr,
		witness_num:        C.ulong(witnessAddrBuffLen),
		input:              (*C.uchar)((unsafe.Pointer)(&contract.Args[0])),
		input_len:          C.ulong(len(contract.Args)),
		wasmvm_service_ptr: C.ulonglong(this.wasmVmServicePtr),
		gas_left:           C.ulonglong(0),
		call_output:        (*C.uchar)((unsafe.Pointer)(&contract.Args[0])),
		call_output_len:    C.ulong(0),
	}

	fmt.Printf("wasm invoke 00000\n")
	C.ontio_call_invoke((*C.uchar)((unsafe.Pointer)(&wasmCode[0])), C.uint(len(wasmCode)), inter_chain)
	fmt.Printf("wasm invoke 11111\n")
	host := &Runtime{Service: this, Input: contract.Args}

	//var compiled *exec.CompiledModule
	//if CodeCache != nil {
	//	cached, ok := CodeCache.Get(contract.Address.ToHexString())
	//	if ok {
	//		compiled = cached.(*exec.CompiledModule)
	//	}
	//}

	//if compiled == nil {
	//	compiled, err = ReadWasmModule(wasmCode, false)
	//	if err != nil {
	//		return nil, err
	//	}
	//	CodeCache.Add(contract.Address.ToHexString(), compiled)
	//}

	//vm, err := exec.NewVMWithCompiled(compiled, WASM_MEM_LIMITATION)
	//if err != nil {
	//	return nil, VM_INIT_FAULT
	//}

	//vm.HostData = host

	//vm.AvaliableGas = &exec.Gas{GasLimit: this.GasLimit, LocalGasCounter: 0, GasPrice: this.GasPrice, GasFactor: this.GasFactor, ExecStep: this.ExecStep}
	//vm.CallStackDepth = uint32(WASM_CALLSTACK_LIMIT)
	//vm.RecoverPanic = true

	//entryName := CONTRACT_METHOD_NAME

	//entry, ok := compiled.RawModule.Export.Entries[entryName]

	//if ok == false {
	//	return nil, errors.NewErr("[Call]Method:" + entryName + " does not exist!")
	//}

	////get entry index
	//index := int64(entry.Index)

	////get function index
	//fidx := compiled.RawModule.Function.Types[int(index)]

	////get  function type
	//ftype := compiled.RawModule.Types.Entries[int(fidx)]

	////no returns of the entry function
	//if len(ftype.ReturnTypes) > 0 {
	//	return nil, errors.NewErr("[Call]ExecCode error! Invoke function sig error")
	//}

	////no args for passed in, all args in runtime input buffer
	//this.vm = vm

	//_, err = vm.ExecCode(index)

	//if err != nil {
	//	return nil, errors.NewErr("[Call]ExecCode error!" + err.Error())
	//}

	//pop the current context
	this.ContextRef.PopContext()

	return host.Output, nil
}
