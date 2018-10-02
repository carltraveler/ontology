package makemap

import (
	"fmt"
	vm "github.com/ontio/ontology/vm/neovm"
	"github.com/ontio/ontology/vm/neovm/utils"
)

type OpCode byte

var Codemap map[vm.OpCode]string

func Dumpcode(code []byte) {
	var OpReader *utils.VmReader
	for true {
		OpReader = utils.NewVmReader(code)
		OpName, err := OpReader.ReadByte()
		if err != nil {
			fmt.Printf("READByte ERROR\n")
			return
		}
		OP := vm.OpCode(OpName)

		position := OpReader.Position()
		fmt.Printf("Position:%d, len:%d\n", position, len(code))
		if position >= len(code) {
			break
		}

		fmt.Printf("offset: %d  OpCode:%d  OpName:", OpReader.Position(), OP)
		if OP >= vm.PUSHBYTES1 && OP <= vm.PUSHBYTES75 {
			fmt.Printf("%s%d\n", "PUSHBYTES", OP)
		} else {
			fmt.Printf("%s\n", Codemap[OP])
		}
	}
	return
}

//func Makemap() {}
func init() {
	print("init makemap xxxxxxxxxxxxxxxxxx\n")
	Codemap = map[vm.OpCode]string{
		0x00: "PUSH0",
		0x01: "PUSHBYTES1",
		0x4B: "PUSHBYTES75",
		0x4C: "PUSHDATA1",
		0x4D: "PUSHDATA2",
		0x4E: "PUSHDATA4",
		0x4F: "PUSHM1",
		0x51: "PUSH1",
		0x52: "PUSH2",
		0x53: "PUSH3",
		0x54: "PUSH4",
		0x55: "PUSH5",
		0x56: "PUSH6",
		0x57: "PUSH7",
		0x58: "PUSH8",
		0x59: "PUSH9",
		0x5A: "PUSH10",
		0x5B: "PUSH11",
		0x5C: "PUSH12",
		0x5D: "PUSH13",
		0x5E: "PUSH14",
		0x5F: "PUSH15",
		0x60: "PUSH16",
		0x61: "NOP",
		0x62: "JMP",
		0x63: "JMPIF",
		0x64: "JMPIFNOT",
		0x65: "CALL",
		0x66: "RET",
		0x67: "APPCALL",
		0x68: "SYSCALL",
		0x69: "TAILCALL",
		0x6A: "DUPFROMALTSTACK",
		0x6B: "TOALTSTACK",
		0x6C: "FROMALTSTACK",
		0x6D: "XDROP",
		0x72: "XSWAP",
		0x73: "XTUCK",
		0x74: "DEPTH",
		0x75: "DROP",
		0x76: "DUP",
		0x77: "NIP",
		0x78: "OVER",
		0x79: "PICK",
		0x7A: "ROLL",
		0x7B: "ROT",
		0x7C: "SWAP",
		0x7D: "TUCK",
		0x7E: "CAT",
		0x7F: "SUBSTR",
		0x80: "LEFT",
		0x81: "RIGHT",
		0x82: "SIZE",
		0x83: "INVERT",
		0x84: "AND",
		0x85: "OR",
		0x86: "XOR",
		0x87: "EQUAL",
		0x8B: "INC",
		0x8C: "DEC",
		0x8D: "SIGN",
		0x8F: "NEGATE",
		0x90: "ABS",
		0x91: "NOT",
		0x92: "NZ",
		0x93: "ADD",
		0x94: "SUB",
		0x95: "MUL",
		0x96: "DIV",
		0x97: "MOD",
		0x98: "SHL",
		0x99: "SHR",
		0x9A: "BOOLAND",
		0x9B: "BOOLOR",
		0x9C: "NUMEQUAL",
		0x9E: "NUMNOTEQUAL",
		0x9F: "LT",
		0xA0: "GT",
		0xA1: "LTE",
		0xA2: "GTE",
		0xA3: "MIN",
		0xA4: "MAX",
		0xA5: "WITHIN",
		0xA7: "SHA1",
		0xA8: "SHA256",
		0xA9: "HASH160",
		0xAA: "HASH256",
		0xAC: "CHECKSIG",
		0xAD: "VERIFY",
		0xAE: "CHECKMULTISIG",
		0xC0: "ARRAYSIZE",
		0xC1: "PACK",
		0xC2: "UNPACK",
		0xC3: "PICKITEM",
		0xC4: "SETITEM",
		0xC5: "NEWARRAY",
		0xC6: "NEWSTRUCT",
		0xC7: "NEWMAP",
		0xC8: "APPEND",
		0xC9: "REVERSE",
		0xCA: "REMOVE",
		0xCB: "HASKEY",
		0xCC: "KEYS",
		0xCD: "VALUES",
		0xF0: "THROW",
		0xF1: "THROWIFNOT"}
}
