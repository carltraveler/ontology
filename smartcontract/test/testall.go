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
	testdir := "/home/steven/github/neo-boa/testdata/"
	codeFile := [...]string{"funcall.avm", "arrayreverse.avm", "IterTest5.avm", "IterTest6.avm"}

	for i := 0; i < len(codeFile); i++ {
		fmt.Printf("Running file : %s\n", codeFile[i])
		CheckExist, err := pathExists(testdir + codeFile[i])
		if err != nil {
			fmt.Printf("File:%s. ERROR: %s", codeFile[i], err)
			continue
		}

		if CheckExist == false {
			fmt.Printf("File:%s not exist\n")
			continue
		}

		codeStr, err := ioutil.ReadFile(testdir + codeFile[i])

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
			Config:     config,
			Gas:        10000,
			CloneCache: nil,
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
}
