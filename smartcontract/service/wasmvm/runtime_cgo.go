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
	"unsafe"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
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
func jitRreadWasmMemory(vmctx *C.uchar, data_ptr uint32, data_len uint32) ([]byte, C.Cgoerror) {
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

func jitErr(err error) C.Cgoerror {
	s := err.Error()
	ptr := C.CBytes([]byte(s))
	l := len(s)
	cgoerr := C.ontio_err_from_cstring((*C.uchar)(ptr), (C.uint)(l))
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
	err := C.ontio_set_calloutput(vmctx, (*C.uchar)((unsafe.Pointer)(&result[0])), C.uint(len(result)))
	return err
}

// c to call go interface

//export ontio_debug_cgo
func ontio_debug_cgo(vmctx *C.uchar, data_ptr uint32, data_len uint32) C.Cgoerror {
	fmt.Printf("ontio_debug_cgo enter\n")
	bs, err := jitRreadWasmMemory(vmctx, data_ptr, data_len)

	if uint32(err.err) != 0 {
		return err
	}
	log.Infof("[WasmContract]Debug:%s\n", bs)

	return C.Cgoerror{err: 0} //true
}

//export ontio_call_contract_cgo
func ontio_call_contract_cgo(vmctx *C.uchar, contractAddr uint32, inputPtr uint32, inputLen uint32) C.Cgoerror {
	var contractAddress common.Address

	Service, err := jitService(vmctx)
	if uint32(err.err) != 0 {
		return err
	}

	buff, err := jitRreadWasmMemory(vmctx, contractAddr, 20)
	if uint32(err.err) != 0 {
		return err
	}

	copy(contractAddress[:], buff[:])

	inputs, err := jitRreadWasmMemory(vmctx, inputPtr, inputLen)
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

	setCallOutPut(vmctx, result)

	return C.Cgoerror{err: 0}
}

// call to c
func invokeJit(this *WasmVmService, contract *states.WasmContractParam, wasmCode []byte) ([]byte, error) {
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
	output := C.ontio_call_invoke((*C.uchar)((unsafe.Pointer)(&wasmCode[0])), C.uint(len(wasmCode)), inter_chain)
	if output.err != 0 {
		err := errors.NewErr(C.GoString((*C.char)((unsafe.Pointer)(output.errmsg))))
		C.ontio_free_cgooutput(output)
		return nil, err
	}

	fmt.Printf("wasm invoke 11111\n")
	outputbuffer := C.GoBytes((unsafe.Pointer)(output.output), (C.int)(output.outputlen))
	C.ontio_free_cgooutput(output)
	return outputbuffer, nil
}
