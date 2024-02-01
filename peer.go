package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"p2p/colorize"
	"strings"
)

var Username string

func main() {
	var myAddr string
	flag.StringVar(&myAddr, "listen", "", "address which server will be listening")
	flag.Parse()
	if myAddr == "" {
		fmt.Println(colorize.Colorize(3, "missing '-listen' argument"))
		os.Exit(1)
	}

	for {
		fmt.Print("Ваше имя: ")
		Username, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		Username = strings.Join(strings.Fields(Username), " ")
		if Username != "" {
			break
		}
		fmt.Println("Некорректное имя")
	}

	ln, err := net.Listen("tcp", myAddr)
	if err != nil {
		fmt.Println(colorize.Colorize(3, err.Error()))
		os.Exit(1)
	}

	go broadcaster()

	go handleInput()

	go connectArgs()
	listen(ln)
}

func connectArgs() {
	for _, addr := range flag.Args() {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			fmt.Println(colorize.Colorize(3, err.Error()))
			continue
		}
		go handleOutcoming(conn)
	}
}

func handleInput() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		messages <- Message{colorize.Colorize(4, Username) + " -> " + s.Text(), nil}
	}
}

func listen(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(colorize.Colorize(3, err.Error()))
			continue
		}
		go handleIncoming(conn)
	}
}

type Message struct {
	text string
	ch   chan string
}

func handleIncoming(conn net.Conn) {
	handleConn(conn, true)
}

func handleOutcoming(conn net.Conn) {
	handleConn(conn, false)
}

func handleConn(conn net.Conn, isIncoming bool) {
	ch := make(chan string)
	go clientWriter(conn, ch)

	fmt.Fprintln(conn, Username)

	who, _ := bufio.NewReader(conn).ReadString('\n')
	who = strings.TrimSpace(who)

	if isIncoming {
		messages <- Message{colorize.Colorize(2, who+" подключился"), ch}
		fmt.Fprintln(os.Stdout, colorize.Colorize(2, who+" подключился"))
	}

	entering <- ch

	s := bufio.NewScanner(conn)
	for s.Scan() {
		fmt.Fprintln(os.Stdout, s.Text())
		messages <- Message{s.Text(), ch}
	}

	leaving <- ch
	messages <- Message{colorize.Colorize(1, who+" отключился"), nil}
	fmt.Fprintln(os.Stdout, colorize.Colorize(1, who+" отключился"))
	conn.Close()
}

type client chan<- string

var (
	messages = make(chan Message)
	entering = make(chan client)
	leaving  = make(chan client)
)

func broadcaster() {
	var clients = make(map[client]bool)
	for {
		select {
		case msg := <-messages:
			for cli := range clients {
				if msg.ch != cli {
					cli <- msg.text
				}
			}
		case cli := <-entering:
			clients[cli] = true
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}
