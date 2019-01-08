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
	"os"
	//"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	. "github.com/ontio/ontology/smartcontract"
	"github.com/ontio/ontology/smartcontract/test/makemap"
	"io/ioutil"
	//"os"
	//"strings"
)

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func main() {
	makemap.DEBUGMODE_MAP = false
	testdir := "/home/cl/github/ontology-python-compiler/testdata/test/"
	codeFile := [...]string{"funcall", "IterTest5", "IterTest6", "test_append_remove", "test_boolop", "test_call_and_if", "test_compare", "test_dict0", "test_dict1", "test_dict_com0", "test_dict_com1", "test_dict_com2", "test_dict", "test_equal_not", "test_Fibonacci", "test_global", "test_list_com", "test_slice", "test_state", "test_while1", "test_while2", "test_while", "ifexpr", "test_for", "BinopTest", "test_elt_in", "isnot", "test_reverse_reversed", "str2int", "test_split", "test_hex2byte_and_bytereverse", "str2int", "test_lil", "test_bytes2str", "test_div", "test_str", "test_boolop_1", "test_boolop_origin", "test_compare_1", "test_for_1", "test_while_3", "test_in"}
	extendname := ".avm"

	for i := 0; i < len(codeFile); i++ {
		codefile_run := codeFile[i] + extendname
		fmt.Printf("Running file : %s\n", codefile_run)
		CheckExist, err := pathExists(testdir + codefile_run)
		if err != nil {
			fmt.Printf("File:%s. ERROR: %s", codefile_run)
			continue
		}

		if CheckExist == false {
			fmt.Printf("File:%s not exist\n")
			continue
		}

		codeStr, err := ioutil.ReadFile(testdir + codefile_run)

		if err != nil {
			fmt.Errorf("Please specify code file.")
			return
		}

		evilBytecode := codeStr
		config := &Config{
			Time:   10,
			Height: 10,
			Tx:     &types.Transaction{},
		}

		sc := SmartContract{
			Config:  config,
			Gas:     1000000000,
			CacheDB: nil,
		}

		engine, err := sc.NewExecuteEngine(evilBytecode)

		if err != nil {
			fmt.Printf("0ERROR runned: %s\n", err)
			return
		}
		_, err = engine.Invoke()

		if err != nil {
			fmt.Printf("1ERROR runned: %s\n", err)
			return
		}

		print("done\n")
	}
	print("all testbench run ok\n")
}
