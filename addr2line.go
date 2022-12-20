	/*
	 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	 *
	 *   Name: nav - Kernel source code analysis tool
	 *   Description: Extract call trees for kernel API
	 *
	 *   Author: Alessandro Carminati <acarmina@redhat.com>
	 *   Author: Maurizio Papini <mpapini@redhat.com>
	 *
	 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	 *
	 *   Copyright (c) 2008-2010 Red Hat, Inc. All rights reserved.
	 *
	 *   This copyrighted material is made available to anyone wishing
	 *   to use, modify, copy, or redistribute it subject to the terms
	 *   and conditions of the GNU General Public License version 2.
	 *
	 *   This program is distributed in the hope that it will be
	 *   useful, but WITHOUT ANY WARRANTY; without even the implied
	 *   warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
	 *   PURPOSE. See the GNU General Public License for more details.
	 *
	 *   You should have received a copy of the GNU General Public
	 *   License along with this program; if not, write to the Free
	 *   Software Foundation, Inc., 51 Franklin Street, Fifth Floor,
	 *   Boston, MA 02110-1301, USA.
	 *
	 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	 */
package main

import (
	"fmt"
	"strings"
	"database/sql"
	"path/filepath"
	"sync"
	addr2line "github.com/elazarl/addr2line"
)

type workloads struct{
	Addr	uint64
	Name	string
	Query	string
	DB	*sql.DB
	}

type Addr2line_items struct {
	Addr		uint64
	File_name	string
	}

type ins_f func(*sql.DB, string, bool)

var Addr2line_cache []Addr2line_items;
var mu sync.Mutex

func addr2line_init(fn string) (*addr2line.Addr2line, chan workloads){
	a, err := addr2line.New(fn)
	if err != nil {
		panic( err)
		}
	adresses := make(chan workloads, 16)
	go workload(a, adresses, Insert_data)
	return a, adresses
}
func in_cache(Addr uint64, Addr2line_cache []Addr2line_items)(bool, string){
	for _,a := range Addr2line_cache {
		if a.Addr == Addr {
			return true, a.File_name
			}
		}
	return false, ""
}

func resolve_addr(a *addr2line.Addr2line, address uint64) string{
	var res string = ""
//m1 := regexp.MustCompile(`^([^ ]*) .*`)
//fmt.Println(m1.ReplaceAllString("/home/alessandro/src/linux-5.18.4/./arch/x86/kernel/../include/asm/trace/./irq_vectors.h:41 (discriminator 6)", "$1"))
	mu.Lock()
	rs, _ := a.Resolve(address)
	mu.Unlock()
	if len(rs)==0 {
		res="NONE"
		}
	for _, a:=range rs{
		res=fmt.Sprintf("%s:%d",filepath.Clean(a.File), a.Line)
		}
	return  res
}

func workload(a *addr2line.Addr2line, addresses chan workloads, insert_func ins_f){
	var e	workloads
	var qready string

	for {
		e = <-addresses
		switch e.Name {
		case "None":
			insert_func(e.DB, e.Query, false)
			break
		default:
			mu.Lock()
			rs, _ := a.Resolve(e.Addr)
			mu.Unlock()
			if len(rs)==0 {
				qready=fmt.Sprintf(e.Query, "NONE")
				}
			for _, a:=range rs{
				qready=fmt.Sprintf(e.Query, filepath.Clean(a.File))
				if a.Function == strings.ReplaceAll(e.Name, "sym.", "") {
					break
					}
				}
			insert_func(e.DB, qready, false)
			break
			}
	}
}

func spawn_query(db *sql.DB, addr uint64, name string, addresses chan workloads, query string) {
	addresses <- workloads{addr, name, query, db}
}
