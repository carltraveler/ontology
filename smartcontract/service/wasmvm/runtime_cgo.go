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
	"unsafe"

	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/smartcontract/states"
)

// c to call go interface

//export ontio_debug_cgo
func ontio_debug_cgo(vmctx *C.uchar, data_ptr uint32, data_len uint32) uint32 {
	fmt.Printf("DebugCgo enter\n")
	bs := make([]byte, data_len)
	err := C.ontio_read_wasmvm_memory(vmctx, (*C.uchar)(unsafe.Pointer(&bs[0])), C.uint(data_ptr), C.uint(data_len))

	if uint32(err) == 0 {
		return 0 //false
	}
	log.Infof("[WasmContract]Debug:%s\n", bs)

	return 1 //true
}

// call to c
func invokeJit(this *WasmVmService, contract *states.WasmContractParam, wasmCode []byte) {
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
}
