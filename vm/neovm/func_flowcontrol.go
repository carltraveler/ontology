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

package neovm

import (
	"github.com/ontio/ontology/vm/neovm/errors"
)

func opNop(e *ExecutionEngine) (VMState, error) {
	return NONE, nil
}

func opJmp(e *ExecutionEngine) (VMState, error) { // jmp offset is saved in avm jump OPCODE next 2 bytes
	num, err := e.Context.OpReader.ReadInt16()
	if err != nil {
		return FAULT, err
	}
	offset := int(num)

	offset = e.Context.GetInstructionPointer() + offset - 3 // calculate the target addr, affset is not a relative address, but a absolute address. so must bigger than 0

	if offset < 0 || offset > len(e.Context.Code) { // offset < 0 indicate can not jmp back? why?
		return FAULT, errors.ERR_FAULT
	}
	var fValue = true

	if e.OpCode > JMP { // if jump is not absolute JMP. it JMPIF or JMPIFNOT
		if EvaluationStackCount(e) < 1 {
			return FAULT, errors.ERR_UNDER_STACK_LEN
		}
		var err error
		fValue, err = PopBoolean(e) // JMPIF JMPIFNOT have a Boolean arg in stack
		if err != nil {
			return FAULT, err
		}
		if e.OpCode == JMPIFNOT {
			fValue = !fValue
		}
	}

	if fValue { //if fValue not set the new pc(offset) is not set to vm
		e.Context.SetInstructionPointer(int64(offset)) // so InstructionPointer is always indicate the next instr to exec. when in OPEXEC code. the position is must be the nest instr addr
	}
	return NONE, nil
}

func opCall(e *ExecutionEngine) (VMState, error) {
	context := e.Context.Clone()
	e.Context.SetInstructionPointer(int64(e.Context.GetInstructionPointer() + 2)) // set the old context's return InstructionPointer. due after CALL avm have 2 bytes. so return addr need +2
	e.OpCode = JMP                                                                // absolute JMP
	e.PushContext(context)                                                        // PushContext will update e.Context to the new context. note, the new context's context InstructionPointer was not changed (first cloned). just update the CALL opcode ===> JMP Opcode. so after 2 bytes is for the JMP opcode offset.
	return opJmp(e)
}

func opDCALL(e *ExecutionEngine) (VMState, error) {
	context := e.Context.Clone()
	e.Context.SetInstructionPointer(int64(e.Context.GetInstructionPointer()))
	e.PushContext(context)

	dest, err := PopBigInt(e)
	if err != nil {
		return FAULT, errors.ERR_DCALL_OFFSET_ERROR
	}
	target := dest.Int64()

	if target < 0 || int(target) >= len(e.Context.Code) {
		return FAULT, errors.ERR_DCALL_OFFSET_ERROR
	}

	e.Context.SetInstructionPointer(target)

	return NONE, nil
}

func opRet(e *ExecutionEngine) (VMState, error) {
	e.PopContext() // will update the current context to prev Context which saved the prev postion after opCALL + 3 bytes.
	return NONE, nil
}
