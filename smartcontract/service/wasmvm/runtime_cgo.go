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
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"io"
	"math"
	"unsafe"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	//"github.com/ontio/ontology/smartcontract/context"
	states2 "github.com/ontio/ontology/core/states"
	"github.com/ontio/ontology/smartcontract/event"
	native2 "github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
	"github.com/ontio/ontology/smartcontract/service/util"
	"github.com/ontio/ontology/smartcontract/states"
	"github.com/ontio/ontology/vm/crossvm_codec"
	neotypes "github.com/ontio/ontology/vm/neovm/types"
)

func getContractType(Service *WasmVmService, addr common.Address) (ContractType, error) {
	if utils.IsNativeContract(addr) {
		return NATIVE_CONTRACT, nil
	}

	dep, err := Service.CacheDB.GetContract(addr)
	if err != nil {
		return UNKOWN_CONTRACT, err
	}
	if dep == nil {
		return UNKOWN_CONTRACT, errors.NewErr("contract is not exist.")
	}
	if dep.VmType() == payload.WASMVM_TYPE {
		return WASMVM_CONTRACT, nil
	}

	return NEOVM_CONTRACT, nil
}

// common interface
func jitReadWasmMemory(vmctx *C.uchar, data_ptr uint32, data_len uint32) ([]byte, C.Cgoerror) {
	if data_len == 0 {
		return []byte{}, C.Cgoerror{err: 0}
	}

	cgobuffer := C.ontio_read_wasmvm_memory(vmctx, C.uint(data_ptr), C.uint(data_len))
	if cgobuffer.err != 0 {
		return nil, C.Cgoerror{
			err:    cgobuffer.err,
			errmsg: cgobuffer.errmsg,
		}
	}

	buff := C.GoBytes((unsafe.Pointer)(cgobuffer.output), (C.int)(cgobuffer.outputlen))
	C.ontio_memfree(cgobuffer.output)
	return buff, C.Cgoerror{err: 0}
}

func jitWriteWasmMemory(vmctx *C.uchar, p []byte, off uint32) C.Cgou32 {
	if len(p) != 0 {
		return C.ontio_write_wasmvm_memory(vmctx, (*C.uchar)((unsafe.Pointer)(&p[0])), C.uint(off), C.uint(len(p)))
	}

	return C.Cgou32{v: 0, err: 0}
}

func jitErr(err error) C.Cgoerror {
	s := err.Error()
	ptr := C.CBytes([]byte(s))
	l := len(s)
	cgoerr := C.ontio_error((*C.uchar)(ptr), (C.uint)(l))
	C.free(ptr)
	return cgoerr
}

func jitService(vmctx *C.uchar) (*WasmVmService, C.Cgoerror) {
	cgou64 := C.ontio_wasm_service_ptr(vmctx)
	if cgou64.err != 0 {
		return nil, C.Cgoerror{
			err:    1,
			errmsg: cgou64.errmsg,
		}
	}

	return GetWasmVmService(uint64(cgou64.v)), C.Cgoerror{err: 0}
}

func setCallOutPut(vmctx *C.uchar, result []byte) C.Cgoerror {
	var err C.Cgoerror
	if len(result) != 0 {
		err = C.ontio_set_calloutput(vmctx, (*C.uchar)((unsafe.Pointer)(&result[0])), C.uint(len(result)))
	} else {
		// when call native. zero len bytes of result consider as 0.
		err = C.ontio_set_calloutput(vmctx, (*C.uchar)((unsafe.Pointer)(nil)), C.uint(0))
	}

	return err
}

// c to call go interface

//export ontio_contract_create_cgo
func ontio_contract_create_cgo(vmctx *C.uchar,
	codePtr uint32,
	codeLen uint32,
	vmType uint32,
	namePtr uint32,
	nameLen uint32,
	verPtr uint32,
	verLen uint32,
	authorPtr uint32,
	authorLen uint32,
	emailPtr uint32,
	emailLen uint32,
	descPtr uint32,
	descLen uint32,
	newAddressPtr uint32) C.Cgou32 {
	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	code, err := jitReadWasmMemory(vmctx, codePtr, codeLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	//cost := CONTRACT_CREATE_GAS + uint64(uint64(codeLen)/PER_UNIT_CODE_LEN)*UINT_DEPLOY_CODE_LEN_GAS

	name, err := jitReadWasmMemory(vmctx, namePtr, nameLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	version, err := jitReadWasmMemory(vmctx, verPtr, verLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	author, err := jitReadWasmMemory(vmctx, authorPtr, authorLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	email, err := jitReadWasmMemory(vmctx, emailPtr, emailLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	desc, err := jitReadWasmMemory(vmctx, descPtr, descLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	dep, errs := payload.CreateDeployCode(code, vmType, name, version, author, email, desc)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	wasmCode, errs := dep.GetWasmCode()
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}
	_, errs = ReadWasmModule(wasmCode, true)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	contractAddr := dep.Address()

	item, errs := Service.CacheDB.GetContract(contractAddr)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	if item != nil {
		err := jitErr(errors.NewErr("contract has been deployed"))
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	Service.CacheDB.PutContract(dep)

	cgou32 := jitWriteWasmMemory(vmctx, contractAddr[:], newAddressPtr)

	return cgou32
}

//export ontio_contract_migrate_cgo
func ontio_contract_migrate_cgo(vmctx *C.uchar,
	codePtr uint32,
	codeLen uint32,
	vmType uint32,
	namePtr uint32,
	nameLen uint32,
	verPtr uint32,
	verLen uint32,
	authorPtr uint32,
	authorLen uint32,
	emailPtr uint32,
	emailLen uint32,
	descPtr uint32,
	descLen uint32,
	newAddressPtr uint32) C.Cgou32 {
	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	code, err := jitReadWasmMemory(vmctx, codePtr, codeLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	//cost := CONTRACT_CREATE_GAS + uint64(uint64(codeLen)/PER_UNIT_CODE_LEN)*UINT_DEPLOY_CODE_LEN_GAS

	name, err := jitReadWasmMemory(vmctx, namePtr, nameLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	version, err := jitReadWasmMemory(vmctx, verPtr, verLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	author, err := jitReadWasmMemory(vmctx, authorPtr, authorLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	email, err := jitReadWasmMemory(vmctx, emailPtr, emailLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	desc, err := jitReadWasmMemory(vmctx, descPtr, descLen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	dep, errs := payload.CreateDeployCode(code, vmType, name, version, author, email, desc)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	wasmCode, errs := dep.GetWasmCode()
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}
	_, errs = ReadWasmModule(wasmCode, true)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	contractAddr := dep.Address()

	item, errs := Service.CacheDB.GetContract(contractAddr)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	if item != nil {
		err := jitErr(errors.NewErr("contract has been deployed"))
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	oldAddress := Service.ContextRef.CurrentContext().ContractAddress

	Service.CacheDB.PutContract(dep)
	Service.CacheDB.DeleteContract(oldAddress)

	iter := Service.CacheDB.NewIterator(oldAddress[:])
	for has := iter.First(); has; has = iter.Next() {
		key := iter.Key()
		val := iter.Value()

		newkey := serializeStorageKey(contractAddr, key[20:])

		Service.CacheDB.Put(newkey, val)
		Service.CacheDB.Delete(key)
	}

	iter.Release()
	if errs := iter.Error(); errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	cgou32 := jitWriteWasmMemory(vmctx, contractAddr[:], newAddressPtr)

	return cgou32
}

//export ontio_contract_destroy_cgo
func ontio_contract_destroy_cgo() C.Cgoerror {
	return C.Cgoerror{err: 0}
}

//export ontio_storage_read_cgo
func ontio_storage_read_cgo(vmctx *C.uchar, keyPtr uint32, klen uint32, val uint32, vlen uint32, offset uint32) C.Cgou32 {
	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	keybytes, err := jitReadWasmMemory(vmctx, keyPtr, klen)
	if uint32(err.err) != 0 {
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	key := serializeStorageKey(Service.ContextRef.CurrentContext().ContractAddress, keybytes)

	raw, errs := Service.CacheDB.Get(key)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	if raw == nil {
		return C.Cgou32{v: C.uint(math.MaxUint32), err: 0}
	}

	item, errs := states2.GetValueFromRawStorageItem(raw)
	if errs != nil {
		err := jitErr(errs)
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	length := vlen
	itemlen := uint32(len(item))
	if itemlen < vlen {
		length = itemlen
	}

	if uint32(len(item)) < offset {
		err := jitErr(errors.NewErr("offset is invalid"))
		return C.Cgou32{v: 0, err: 1, errmsg: err.errmsg}
	}

	cgou32 := jitWriteWasmMemory(vmctx, item[offset:offset+length], val)
	if uint32(cgou32.err) != 0 {
		return cgou32
	}

	return C.Cgou32{v: C.uint(len(item)), err: 0}
}

//export ontio_storage_write_cgo
func ontio_storage_write_cgo(vmctx *C.uchar, keyPtr uint32, keyLen uint32, valPtr uint32, valLen uint32) C.Cgoerror {
	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return err
	}

	keybytes, err := jitReadWasmMemory(vmctx, keyPtr, keyLen)
	if uint32(err.err) != 0 {
		return err
	}

	valbytes, err := jitReadWasmMemory(vmctx, valPtr, valLen)
	if uint32(err.err) != 0 {
		return err
	}

	//cost := uint64((len(keybytes)+len(valbytes)-1)/1024+1) * STORAGE_PUT_GAS
	//self.checkGas(cost)

	key := serializeStorageKey(Service.ContextRef.CurrentContext().ContractAddress, keybytes)

	Service.CacheDB.Put(key, states2.GenRawStorageItem(valbytes))

	return C.Cgoerror{err: 0}
}

//export ontio_storage_delete_cgo
func ontio_storage_delete_cgo(vmctx *C.uchar, keyPtr uint32, keyLen uint32) C.Cgoerror {
	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return err
	}

	//self.checkGas(STORAGE_DELETE_GAS)

	keybytes, err := jitReadWasmMemory(vmctx, keyPtr, keyLen)
	if uint32(err.err) != 0 {
		return err
	}

	key := serializeStorageKey(Service.ContextRef.CurrentContext().ContractAddress, keybytes)

	Service.CacheDB.Delete(key)

	return C.Cgoerror{err: 0}
}

//export ontio_notify_cgo
func ontio_notify_cgo(vmctx *C.uchar, ptr uint32, l uint32) C.Cgoerror {
	if l >= neotypes.MAX_NOTIFY_LENGTH {
		return jitErr(errors.NewErr("notify length over the uplimit"))
	}

	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return err
	}

	bs, err := jitReadWasmMemory(vmctx, ptr, l)
	if uint32(err.err) != 0 {
		return err
	}

	notify := &event.NotifyEventInfo{ContractAddress: Service.ContextRef.CurrentContext().ContractAddress}
	val := crossvm_codec.DeserializeNotify(bs)
	notify.States = val

	notifys := make([]*event.NotifyEventInfo, 1)
	notifys[0] = notify
	Service.ContextRef.PushNotifications(notifys)

	return C.Cgoerror{err: 0}
}

//export ontio_debug_cgo
func ontio_debug_cgo(vmctx *C.uchar, data_ptr uint32, data_len uint32) C.Cgoerror {
	bs, err := jitReadWasmMemory(vmctx, data_ptr, data_len)
	if uint32(err.err) != 0 {
		return err
	}

	log.Infof("[WasmContract]Debug:%s\n", bs)
	return C.Cgoerror{err: 0}
}

//export ontio_call_contract_cgo
func ontio_call_contract_cgo(vmctx *C.uchar, contractAddr uint32, inputPtr uint32, inputLen uint32) C.Cgoerror {
	var contractAddress common.Address

	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return err
	}

	buff, err := jitReadWasmMemory(vmctx, contractAddr, 20)
	if uint32(err.err) != 0 {
		return err
	}

	copy(contractAddress[:], buff[:])

	inputs, err := jitReadWasmMemory(vmctx, inputPtr, inputLen)
	if uint32(err.err) != 0 {
		return err
	}

	contracttype, errs := getContractType(Service, contractAddress)
	if errs != nil {
		return jitErr(errs)
	}

	var result []byte

	switch contracttype {
	case NATIVE_CONTRACT:
		source := common.NewZeroCopySource(inputs)
		ver, eof := source.NextByte()
		if eof {
			return jitErr(io.ErrUnexpectedEOF)
		}
		method, _, irregular, eof := source.NextString()
		if irregular {
			return jitErr(common.ErrIrregularData)
		}
		if eof {
			return jitErr(io.ErrUnexpectedEOF)
		}

		args, _, irregular, eof := source.NextVarBytes()
		if irregular {
			return jitErr(common.ErrIrregularData)
		}
		if eof {
			return jitErr(io.ErrUnexpectedEOF)
		}

		contract := states.ContractInvokeParam{
			Version: ver,
			Address: contractAddress,
			Method:  method,
			Args:    args,
		}

		native := &native2.NativeService{
			CacheDB:     Service.CacheDB,
			InvokeParam: contract,
			Tx:          Service.Tx,
			Height:      Service.Height,
			Time:        Service.Time,
			ContextRef:  Service.ContextRef,
			ServiceMap:  make(map[string]native2.Handler),
		}

		tmpRes, err := native.Invoke()
		if err != nil {
			return jitErr(errors.NewErr("[nativeInvoke]AppCall failed:" + err.Error()))
		}

		result = tmpRes

	case WASMVM_CONTRACT:
		conParam := states.WasmContractParam{Address: contractAddress, Args: inputs}
		param := common.SerializeToBytes(&conParam)

		newservice, err := Service.ContextRef.NewExecuteEngine(param, types.InvokeWasm)
		if err != nil {
			return jitErr(err)
		}

		tmpRes, err := newservice.Invoke()
		if err != nil {
			return jitErr(err)
		}

		result = tmpRes.([]byte)

	case NEOVM_CONTRACT:
		evalstack, err := util.GenerateNeoVMParamEvalStack(inputs)
		if err != nil {
			return jitErr(err)
		}

		neoservice, err := Service.ContextRef.NewExecuteEngine([]byte{}, types.InvokeNeo)
		if err != nil {
			return jitErr(err)
		}

		err = util.SetNeoServiceParamAndEngine(contractAddress, neoservice, evalstack)
		if err != nil {
			return jitErr(err)
		}

		tmp, err := neoservice.Invoke()
		if err != nil {
			return jitErr(err)
		}

		if tmp != nil {
			val := tmp.(*neotypes.VmValue)
			source := common.NewZeroCopySink([]byte{byte(crossvm_codec.VERSION)})

			err = neotypes.BuildResultFromNeo(*val, source)
			if err != nil {
				return jitErr(err)
			}
			result = source.Bytes()
		}

	default:
		return jitErr(errors.NewErr("Not a supported contract type"))
	}

	return setCallOutPut(vmctx, result)
}

// call to c
func invokeJit(this *WasmVmService, contract *states.WasmContractParam, wasmCode []byte) ([]byte, error) {
	txHash := this.Tx.Hash()
	witnessAddrBuff, witnessAddrNum := GetAddressBuff(this.Tx.GetSignatureAddresses())
	callersAddrBuff, callersAddrNum := GetAddressBuff(this.ContextRef.GetCallerAddress())
	fmt.Printf("witnessAddrNum : %d\n", witnessAddrNum)
	fmt.Printf("callersAddrNum : %d\n", callersAddrNum)

	var witnessPtr, callersPtr, inputPtr *C.uchar

	if witnessAddrNum == 0 {
		witnessPtr = (*C.uchar)((unsafe.Pointer)(nil))
	} else {
		witnessPtr = (*C.uchar)((unsafe.Pointer)(&witnessAddrBuff[0]))
	}

	if callersAddrNum == 0 {
		callersPtr = (*C.uchar)((unsafe.Pointer)(nil))
	} else {
		callersPtr = (*C.uchar)((unsafe.Pointer)(&callersAddrBuff[0]))
	}

	if len(contract.Args) == 0 {
		inputPtr = (*C.uchar)((unsafe.Pointer)(nil))
	} else {
		inputPtr = (*C.uchar)((unsafe.Pointer)(&contract.Args[0]))
	}

	inter_chain := C.InterOpCtx{
		height:             C.uint(this.Height),
		block_hash:         (*C.uchar)((unsafe.Pointer)(&this.BlockHash[0])),
		timestamp:          C.ulonglong(this.Time),
		tx_hash:            (*C.uchar)((unsafe.Pointer)(&(txHash[0]))),
		self_address:       (*C.uchar)((unsafe.Pointer)(&contract.Address[0])),
		callers:            callersPtr,
		callers_num:        C.ulong(callersAddrNum),
		witness:            witnessPtr,
		witness_num:        C.ulong(witnessAddrNum),
		input:              inputPtr,
		input_len:          C.ulong(len(contract.Args)),
		wasmvm_service_ptr: C.ulonglong(this.wasmVmServicePtr),
		gas_left:           C.ulonglong(0),
	}

	output := C.ontio_call_invoke((*C.uchar)((unsafe.Pointer)(&wasmCode[0])), C.uint(len(wasmCode)), inter_chain)
	defer C.ontio_free_cgooutput(output)

	if output.err != 0 {
		return nil, errors.NewErr(C.GoString((*C.char)((unsafe.Pointer)(output.errmsg))))
	}

	outputbuffer := C.GoBytes((unsafe.Pointer)(output.output), (C.int)(output.outputlen))
	return outputbuffer, nil
}
