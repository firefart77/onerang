package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

var wg sync.WaitGroup

var Connections sync.Map

func main() {
	var myAddr string
	flag.StringVar(&myAddr, "listen", "", "port that server will be listening")
	flag.Parse()
	if myAddr == "" {
		log.Fatal("missing '-listen' flag")
	}
	ln, err := net.Listen("tcp", myAddr)
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	go listen(ln)

	wg.Add(1)
	go WriteLoop()
	for _, addr := range flag.Args() {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Print("cant connect:", err)
			continue
		}
		Connections.Store(conn.RemoteAddr().String(), conn)
		wg.Add(1)
		go ReadConn(conn)
	}
	wg.Wait()
}

func listen(ln net.Listener) {
	defer wg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print("cant connect", err)
			continue
		}
		defer conn.Close()
		Connections.Store(conn.RemoteAddr().String(), conn)
		wg.Add(1)
		go ReadConn(conn)
	}
}

func ReadConn(connFrom net.Conn) {
	defer wg.Done()
	var mes string
	var err error
	r := bufio.NewReader(connFrom)
	for {
		if mes, err = r.ReadString('\n'); err != nil {
			Connections.Delete(connFrom.RemoteAddr().String())
			return
		}
		fmt.Fprint(os.Stdout, mes)
		Connections.Range(func(IP interface{}, conn interface{}) bool {
			if IP != connFrom.RemoteAddr().String() {
				if _, err := fmt.Fprint(conn.(net.Conn), mes); err != nil {
					Connections.Delete(IP.(string))
				}
			}
			return true
		})
	}
}

func WriteLoop() {
	defer wg.Done()
	var mes string
	r := bufio.NewReader(os.Stdin)
	for {
		mes, _ = r.ReadString('\n')
		Connections.Range(func(IP interface{}, conn interface{}) bool {
			if _, err := fmt.Fprint(conn.(net.Conn), mes); err != nil {
				Connections.Delete(IP.(string))
			}
			return true
		})
	}
}
