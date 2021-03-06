// AnonIRCd - Anonymous IRC daemon
// https://github.com/sageru-6ch/anonircd
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"gopkg.in/sorcix/irc.v2"
)

var prefixAnonIRC = irc.Prefix{Name: "AnonIRC"}
var prefixAnonymous = irc.Prefix{Name: "Anonymous", User: "Anon", Host: "IRC"}

const DEFAULT_MOTD = `
  _|_|                                  _|_|_|  _|_|_|      _|_|_|
_|    _|  _|_|_|      _|_|    _|_|_|      _|    _|    _|  _|
_|_|_|_|  _|    _|  _|    _|  _|    _|    _|    _|_|_|    _|
_|    _|  _|    _|  _|    _|  _|    _|    _|    _|    _|  _|
_|    _|  _|    _|    _|_|    _|    _|  _|_|_|  _|    _|    _|_|_|
`
const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const writebuffersize = 10

const (
	CHANNEL_LOBBY  = "#"
	CHANNEL_SERVER = "&"
)

var debugMode = false
var verbose = false

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	var opts struct {
		ConfigFile string `short:"c" long:"config" description:"Configuration file"`
		Debug      int    `short:"d" long:"debug" description:"Serve pprof data on specified port"`
		BareLog    bool   `short:"b" long:"bare-log" description:"Don't add current date/time to log entries"`
		Verbose    bool   `short:"v" long:"verbose" description:"Log verbosely"`
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		log.Panicf("%+v", errors.Wrap(err, "failed to parse flags"))
	}

	if opts.Debug > 0 {
		debugMode = true
		log.Printf("WARNING: Running in debug mode. pprof data is available at http://localhost:%d/debug/pprof/", opts.Debug)
		go http.ListenAndServe(fmt.Sprintf("localhost:%d", opts.Debug), nil)
	}

	if opts.BareLog {
		log.SetFlags(0)
	}

	verbose = opts.Verbose

	s := NewServer(opts.ConfigFile)
	err = s.loadConfig()
	if err != nil {
		log.Panicf("%+v", errors.Wrap(err, "failed to load configuration file"))
	}
	s.connectDatabase()
	defer s.closeDatabase()

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	go func() {
		for {
			<-sighup
			err := s.reload()
			if err != nil {
				log.Println(err)
			}
		}
	}()

	s.listen()
}
