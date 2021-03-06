/*
Copyright 2016 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/urfave/cli"

	"github.com/platform9/fission"
)

func fnCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need --name argument.")
	}

	envName := c.String("env")
	if len(envName) == 0 {
		fatal("Need --env argument.")
	}

	fileName := c.String("code")
	if len(fileName) == 0 {
		fatal("Need --code argument.")
	}

	code, err := ioutil.ReadFile(fileName)
	checkErr(err, fmt.Sprintf("read %v", fileName))

	function := &fission.Function{
		Metadata:    fission.Metadata{Name: fnName},
		Environment: fission.Metadata{Name: envName},
		Code:        string(code),
	}

	_, err = client.FunctionCreate(function)
	checkErr(err, "create function")

	fmt.Printf("function '%v' created\n", fnName)
	return err
}

func fnGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}
	fnUid := c.String("uid")
	m := &fission.Metadata{Name: fnName, Uid: fnUid}

	code, err := client.FunctionGetRaw(m)
	checkErr(err, "get function")

	fmt.Println(string(code))
	return err
}

func fnGetMeta(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	fnUid := c.String("uid")
	m := &fission.Metadata{Name: fnName, Uid: fnUid}

	f, err := client.FunctionGet(m)
	checkErr(err, "get function")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "UID", "ENV")
	fmt.Fprintf(w, "%v\t%v\t%v\n",
		f.Metadata.Name, f.Metadata.Uid, f.Environment.Name)
	w.Flush()
	return err
}

func fnUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")

	function, err := client.FunctionGet(&fission.Metadata{Name: fnName})
	checkErr(err, fmt.Sprintf("read function '%v'", fnName))

	envName := c.String("env")
	if len(envName) > 0 {
		function.Environment.Name = envName
	}

	fileName := c.String("code")
	if len(fileName) > 0 {
		code, err := ioutil.ReadFile(fileName)
		checkErr(err, fmt.Sprintf("read %v", fileName))

		function.Code = string(code)
	}

	_, err = client.FunctionUpdate(function)
	checkErr(err, "update function")

	fmt.Printf("function '%v' updated\n", fnName)
	return err
}

func fnDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	fnUid := c.String("uid")
	m := &fission.Metadata{Name: fnName, Uid: fnUid}

	err := client.FunctionDelete(m)
	checkErr(err, fmt.Sprintf("delete function '%v'", fnName))

	fmt.Printf("function '%v' deleted\n", fnName)
	return err
}

func fnList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fns, err := client.FunctionList()
	checkErr(err, "list functions")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "UID", "ENV")
	for _, f := range fns {
		fmt.Fprintf(w, "%v\t%v\t%v\n",
			f.Metadata.Name, f.Metadata.Uid, f.Environment.Name)
	}
	w.Flush()

	return err
}

func fnEdit(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}
	fnUid := c.String("uid")

	// get function meta
	function, err := client.FunctionGet(&fission.Metadata{Name: fnName, Uid: fnUid})
	checkErr(err, fmt.Sprintf("read function '%v'", fnName))

	// write to tmp file
	tmpFile, err := ioutil.TempFile("", fnName)
	checkErr(err, "create temp file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(function.Code))
	checkErr(err, "write temp file")
	tmpFile.Close()

	// invoke $EDITOR on tmp file and wait for it
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vi"
	}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("%v %v", editor, tmpFile.Name()))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	checkErr(err, "start editor")

	err = cmd.Wait()
	checkErr(err, "wait for editor")

	// read new code out of the file
	contents, err := ioutil.ReadFile(tmpFile.Name())
	checkErr(err, "read temp file")

	function.Code = string(contents)

	// upload the updated function
	newfn, err := client.FunctionUpdate(function)
	checkErr(err, "upload edited function")

	fmt.Printf("function %v updated, new uuid: %v\n", newfn.Name, newfn.Uid)
	return nil
}
