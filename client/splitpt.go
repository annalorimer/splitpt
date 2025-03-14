// Client transport plugin for splitpt
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"

	"sync"
	"syscall"

	spt "anticensorshiptrafficsplitting/splitpt/client/lib"

	"github.com/xtaci/smux"
	pt "gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/goptlib"
)

// Exchanges bytes between SOCKS connection and splitpt connection
// TODO [AHL] This will eventuall have to copy packets to different proxies according
// to the splitting algorithm being used
func copyLoop(socks *pt.SocksConn, sptstream *smux.Stream) {
	done := make(chan struct{}, 2)
	go func() {
		if _, err := io.Copy(socks, sptstream); err != nil {
			log.Printf("[copyLoop] copying to SOCKS resulted in error: %v", err)
		}
		done <- struct{}{}
	}()
	go func() {
		if _, err := io.Copy(sptstream, socks); err != nil {
			log.Printf("[copyLoop] copying from SOCKS resulted in error: %v", err)
			done <- struct{}{}
		}
	}()
	<-done
	log.Printf("copy loop done")
}

func socksAcceptLoop(ln *pt.SocksListener, sptConfig *spt.SplitPTConfig, shutdown chan struct{}, wg *sync.WaitGroup) error {
	log.Printf("socksAcceptLoop()")
	defer ln.Close()
	for {
		conn, err := ln.AcceptSocks()
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Temporary() {
				pt.Log(pt.LogSeverityError, "accept error: "+err.Error())
				continue
			}
		}
		log.Printf("SOCKS accepted %v", conn.Req)
		wg.Add(1)

		go func() {
			defer wg.Done()
			transport, err := spt.NewSplitPTClient(*sptConfig)
			if err != nil {
				log.Printf("Transport error: %s", err)
				conn.Reject()
				return
			}
			log.Printf("Dialing...")
			sconn, err := transport.Dial()
			if err != nil {
				log.Printf("Dial error: %s", err)
				conn.Reject()
				return
			}

			conn.Grant(nil)
			defer sconn.Close()
			copyLoop(conn, sconn)
		}()
	}
	log.Printf("Returning from socksAcceptLoop")
	return nil
}

func handler(conn *pt.SocksConn) error {
	log.Printf("handler()")
	defer conn.Close()
	remote, err := net.Dial("tcp", conn.Req.Target)
	if err != nil {
		conn.Reject()
		log.Printf("Dialing error: %v", err)
		return err
	}
	defer remote.Close()
	err = conn.Grant(remote.RemoteAddr().(*net.TCPAddr))
	if err != nil {
		log.Printf("Connection error: %v", err)
		return err
	}
	// [AHL] do something with conn and remote
	return nil
}

func main() {
	// Parse command line args
	logFilename := flag.String("log", "", "name of log file")
	tomlFilename := flag.String("toml", "", "name of toml config file")
	flag.Parse()

	// Logging
	log.SetFlags(log.LstdFlags | log.LUTC)

	var logOutput = ioutil.Discard
	if *logFilename != "" {
		logFile, err := os.OpenFile(*logFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}
		defer logFile.Close()
		logOutput = logFile
	}
	log.SetOutput(logOutput)

	log.Println("--- Setting up SplitPT ---")
	//var spt.ClientTOMLConfig tomlConfig
	if *tomlFilename == "" {
		log.Printf("toml filename cannot be empty")
		return
	}
	sptConfig, err := spt.GetClientTOMLConfig(*tomlFilename)
	if err != nil {
		log.Printf("Error with toml config: %v", err)
		return
	}
	//log.Println(len(tomlConfig.Connections))
	log.Println("Finished getting config from TOML file")
	log.Println("--- Starting SplitPT ---")

	// splitpt setup

	// begin goptlib client process
	ptInfo, err := pt.ClientSetup(nil)
	if err != nil {
		log.Printf("ClientSetup failed")
		os.Exit(1)
	}

	if ptInfo.ProxyURL != nil {
		pt.ProxyError(fmt.Sprintf("proxy %s is not supported", ptInfo.ProxyURL))
		log.Printf("Proxy is nor supported")
		os.Exit(1)
	}

	listeners := make([]net.Listener, 0)
	shutdown := make(chan struct{})
	var wg sync.WaitGroup

	for _, methodName := range ptInfo.MethodNames {
		switch methodName {
		case "splitpt":
			log.Printf("splitpt method found")
			ln, err := pt.ListenSocks("tcp", "127.0.0.1:0")
			if err != nil {
				pt.CmethodError(methodName, err.Error())
				break
			}
			log.Printf("Started SOCKS listenener at %v", ln.Addr())
			go socksAcceptLoop(ln, sptConfig, shutdown, &wg)
			pt.Cmethod(methodName, ln.Version(), ln.Addr())
			listeners = append(listeners, ln)
		default:
			log.Printf("no such method splitpt")
			pt.CmethodError(methodName, "no such method")
		}
	}
	pt.CmethodsDone()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	if os.Getenv("TOR_PT_EXIT_ON_STIN_CLOSE") == "1" {
		// This environment variable means we should treat EOF on stdin
		// just like SIGTERM: https://bugs.torproject.org/15435
		go func() {
			if _, err := io.Copy(ioutil.Discard, os.Stdin); err != nil {
				log.Printf("calling io.Copy(ioutil.Discard, osStdin) returned error: %v", err)
			}
			log.Printf("synthesizing SIGTERM because of stdin close")
			sigChan <- syscall.SIGTERM
		}()
	}

	// Wait for a signal.
	<-sigChan
	log.Printf("stopping splitpt")

	for _, ln := range listeners {
		ln.Close()
	}
	close(shutdown)
	wg.Wait()
	log.Println("SplitPT is done")

}
