/* Copyright (c) 2016-2017 Gregor Riepl
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import _ "net/http/pprof"
import (
	"log"
	"net/http"
	"runtime"
	"runtime/debug"
)

func EnableProfiling() {
	/*profile, err := os.Create("server.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.WriteHeapProfile(profile)*/
	// Enable block profiling (granularity: 100 ms)
	runtime.SetBlockProfileRate(100000000)
	// Register URL to force reclaiming memory
	http.HandleFunc("/reclaim", func(http.ResponseWriter, *http.Request) {
		log.Printf("Reclaiming memory")
		debug.FreeOSMemory()
	})
	go func() {
		// Start profiling web server
		log.Println(http.ListenAndServe(":6060", nil))
	}()
}
