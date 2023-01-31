/*
 * Copyright (c) 2022 Red Hat, Inc.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

const DBPortNumber = 5432

type argFunc func(*configuration, []string) error

// Command line switch elements.
type cmdLineItems struct {
	Func    argFunc
	Switch  string
	helpStr string
	id      int
	harArg  bool
}

// Represents the application configuration.
type configuration struct {
	DBPassword    string
	LinuxWODebug  string
	StripBin      string
	DBURL         string
	DBUser        string
	LinuxWDebug   string
	DBTargetDB    string
	MaintainersFn string
	KConfigFn     string
	KMakefile     string
	Note          string
	DBPort        int
	Mode          int
}

// Instance of default configuration values.
var defaultConfig = configuration{
	LinuxWDebug:   "vmlinux",
	LinuxWODebug:  "vmlinux.work",
	StripBin:      "/usr/bin/strip",
	DBURL:         "dbs.hqhome163.com",
	DBPort:        DBPortNumber,
	DBUser:        "alessandro",
	DBPassword:    "<password>",
	DBTargetDB:    "kernel_bin",
	MaintainersFn: "MAINTAINERS",
	KConfigFn:     "include/generated/autoconf.h",
	KMakefile:     "Makefile",
	Mode:          enableSymbolsFiles | enableXrefs | enableMaintainers | enableVersionConfig,
	Note:          "upstream",
}

// Inserts a commandline item which is composed by:
// * switch string
// * switch description
// * if the switch requires an additional argument
// * a pointer to the function that manages the switch
// * the configuration that gets updated.
func pushCmdLineItem(switchStr string, helpStr string, hasArg bool, function argFunc, cmdLine *[]cmdLineItems) {
	*cmdLine = append(*cmdLine, cmdLineItems{id: len(*cmdLine) + 1, Switch: switchStr, helpStr: helpStr, harArg: hasArg, Func: function})
}

// This function initializes configuration parser subsystem
// Inserts all the commandline switches supported by the application.
func cmdLineItemInit() []cmdLineItems {
	var res []cmdLineItems

	pushCmdLineItem("-f", "specifies json configuration file", true, funcJconf, &res)
	pushCmdLineItem("-s", "Forces use specified strip binary", true, funcForceStrip, &res)
	pushCmdLineItem("-u", "Forces use specified database userid", true, funcDBUser, &res)
	pushCmdLineItem("-p", "Forces use specified password", true, funcDBPass, &res)
	pushCmdLineItem("-d", "Forces use specified DBHost", true, funcDBHost, &res)
	pushCmdLineItem("-o", "Forces use specified DBPort", true, funcDBPort, &res)
	pushCmdLineItem("-n", "Forces use specified note (default 'upstream')", true, funcNote, &res)
	pushCmdLineItem("-c", "Checks dependencies", false, funcCheck, &res)
	pushCmdLineItem("-h", "This Help", false, funcHelp, &res)

	return res
}

func funcHelp(conf *configuration, fn []string) error {
	return errors.New("dummy")
}

func funcJconf(conf *configuration, fn []string) error {
	jsonFile, err := os.Open(fn[0])
	if err != nil {
		return fmt.Errorf("error while opening json file: %w", err)
	}
	defer func() {
		closeErr := jsonFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	byteValue, _ := io.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, conf)
	if err != nil {
		return fmt.Errorf("error while parsing json file: %w", err)
	}
	return nil
}

func funcForceStrip(conf *configuration, fn []string) error {
	conf.StripBin = fn[0]
	return nil
}

func funcDBUser(conf *configuration, user []string) error {
	conf.DBUser = user[0]
	return nil
}

func funcDBPass(conf *configuration, pass []string) error {
	conf.DBPassword = pass[0]
	return nil
}

func funcDBHost(conf *configuration, host []string) error {
	conf.DBURL = host[0]
	return nil
}

func funcDBPort(conf *configuration, port []string) error {
	s, err := strconv.Atoi(port[0])
	if err != nil {
		return fmt.Errorf("error while parsing port number: %w", err)
	}
	conf.DBPort = s
	return nil
}

func funcNote(conf *configuration, note []string) error {
	conf.Note = note[0]
	return nil
}

func funcCheck(conf *configuration, dummy []string) error {
	return nil
}

// Uses commandline args to generate the help string.
func printHelp(lines []cmdLineItems) {

	for _, item := range lines {
		fmt.Printf(
			"\t%s\t%s\t%s\n",
			item.Switch,
			func(a bool) string {
				if a {
					return "<v>"
				}
				return ""
			}(item.harArg),
			item.helpStr,
		)
	}
}

// Used to parse the command line and generate the command line.
func argsParse(lines []cmdLineItems) (configuration, error) {
	var extra = false
	var conf = defaultConfig
	var f argFunc

	for _, osArg := range os.Args[1:] {
		if !extra {
			for _, arg := range lines {
				if arg.Switch == osArg {
					if arg.harArg {
						f = arg.Func
						extra = true
						break
					}
					err := arg.Func(&conf, []string{})
					if err != nil {
						return defaultConfig, err
					}
				}
			}
			continue
		}
		if extra {
			err := f(&conf, []string{osArg})
			if err != nil {
				return defaultConfig, err
			}
			extra = false
		}
	}
	if extra {
		return conf, errors.New("extra arg needed but none")
	} else {
		return conf, nil
	}
}
