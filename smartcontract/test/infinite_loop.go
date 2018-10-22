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

package main

import (
	"fmt"
	//"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	. "github.com/ontio/ontology/smartcontract"
	"github.com/ontio/ontology/smartcontract/test/makemap"
	"io/ioutil"
	//"os"
	//"strings"
)

func main() {
	//evilBytecode := []byte(" e\xff\u007f\xffhm\xb7%\xa7AAAAAAAAAAAAAAAC\xef\xed\x04INVERT\x95ve")
	//codeFile := "/home/steven/go/src/github.com/ontio/neo-boa/AddTest1.avm"
	//codeFile := "/home/steven/github/neo-boa/AddTest1.avm"
	//codeFile := "/home/steven/github/neo-boa/testA.avm"
	//codeFile := "/home/steven/github/neo-boa/test0.avm"
	//codeFile := "/home/steven/github/neo-boa/lottery.avm"
	//codeFile := "/home/steven/github/neo-boa/test1.avm"
	//codeFile := "/home/steven/github/neo-boa/testdata/min.avm"
	//codeFile := "/home/steven/github/neo-boa/testdata/list_global.avm"
	//codeFile := "/home/steven/github/neo-boa/AppCallTest.avm"
	codeFile := "/home/steven/github/neo-boa/testdata/tmp.avm"

	makemap.DEBUGMODE_MAP = true

	//makemap.Makemap()

	codeStr, err := ioutil.ReadFile(codeFile)
	//fmt.Printf("%x", (codeStr))
	//print("0000000000\n")
	if err != nil {
		fmt.Printf("Read %s Error.\n", codeFile)
		print("xxxxxxxxxxxxxx\n")
		//return nil
		return
	}
	//code := strings.TrimSpace(string(codeStr))
	//fmt.Println(string(code))
	//evilBytecode, err := common.HexToBytes(code)
	evilBytecode := codeStr
	//if err != nil {
	//	fmt.Println("read code:%s error:%s", codeFile, err)
	//	print("yyyyyyyyyyxxxx\n")
	//	return
	//}

	//dbFile := "test"
	//defer func() {
	//	os.RemoveAll(dbFile)
	//}()
	//testLevelDB, err := leveldbstore.NewLevelDBStore(dbFile)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//store := statestore.NewMemDatabase()
	//testBatch := statestore.NewStateStoreBatch(store, testLevelDB)
	config := &Config{
		Time:   10,
		Height: 10,
		Tx:     &types.Transaction{},
	}
	//cache := storage.NewCloneCache(testBatch)
	sc := SmartContract{
		Config:  config,
		Gas:     10000,
		CacheDB: nil,
	}
	engine, err := sc.NewExecuteEngine(evilBytecode)
	if err != nil {
		//t.Fatal(err)
		print("ERROR runned\n")
	}
	_, err = engine.Invoke()

	if err != nil {
		//t.Fatal(err)
		fmt.Printf("ERROR runned: %s\n", err)
		return
	}

	print("all done\n")
}
