package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"

	//"log"
	"net"
	"os"
	"p2p/colorize"
	"strings"
)

var Username string
var ParentIP string

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
		fmt.Println(colorize.Colorize(3, "Некорректное имя"))
	}

	go broadcaster()

	go handleInput()

	if len(flag.Args()) != 0 {
		parentConn, err := net.Dial("tcp", flag.Args()[0])
		if err != nil {
			fmt.Println(colorize.Colorize(3, "cant connect: connection refused"))
		} else {
			go handleOutcoming(parentConn)
		}
	}

	ln, err := net.Listen("tcp", myAddr)
	if err != nil {
		fmt.Println(colorize.Colorize(3, err.Error()))
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(colorize.Colorize(3, err.Error()))
			continue
		}
		go handleIncoming(conn)
	}
}

func handleInput() {
	s := bufio.NewScanner(os.Stdin)
	var text string
	for s.Scan() {
		text = strings.Join(strings.Fields(s.Text()), " ")
		if text == "disconnect" {
			disconnect <- Message{ParentIP: ParentIP}
			break
		} else if text == "" {
			fmt.Println(colorize.Colorize(3, "Нельзя отправить пустое сообщение"))
			continue
		}
		messages <- Message{Text: colorize.Colorize(4, Username) + " -> " + text}
	}
}

type Message struct {
	Text     string
	SenderIP string
	ParentIP string
}

func handleIncoming(conn net.Conn) {
	handleConn(conn, true)
}

func handleOutcoming(conn net.Conn) {
	handleConn(conn, false)
}

func handleConn(conn net.Conn, isIncoming bool) {
	client := NewClient(conn)

	go clientWriter(client)

	fmt.Fprintln(conn, Username)

	who, _ := bufio.NewReader(conn).ReadString('\n')
	who = strings.TrimSpace(who)

	if isIncoming {
		messages <- Message{Text: colorize.Colorize(2, who+" подключился")}
		fmt.Fprintln(os.Stdout, colorize.Colorize(2, who+" подключился"))
	} else {
		ParentIP = client.IP
	}

	entering <- client

	var msg Message
	for {
		if err := client.Dec.Decode(&msg); err != nil {
			fmt.Fprintln(os.Stdout, colorize.Colorize(1, who+" отключился"))
			messages <- Message{Text: colorize.Colorize(1, who+" отключился"), SenderIP: msg.ParentIP}
			leaving <- client

			//log.Print(err)
			break
		}
		if msg.ParentIP != "" {
			messages <- Message{Text: colorize.Colorize(1, who+" отключился"), SenderIP: msg.ParentIP}
			leaving <- client

			newConn, err := net.Dial("tcp", msg.ParentIP)
			if err != nil {
				//log.Print(err)
				continue
			}
			go handleOutcoming(newConn)
			break
		} else {
			fmt.Fprintln(os.Stdout, msg.Text)
			messages <- Message{Text: msg.Text, SenderIP: client.IP}
		}
	}

	conn.Close()
}

type Client struct {
	Ch  chan Message
	Enc *json.Encoder
	Dec *json.Decoder
	IP  string
}

var (
	disconnect = make(chan Message)
	messages   = make(chan Message)
	entering   = make(chan *Client)
	leaving    = make(chan *Client)
)

func broadcaster() {
	var clients = make(map[*Client]bool)
	for {
		select {
		case msg := <-disconnect:
			for cli, _ := range clients {
				if cli.IP != ParentIP {
					cli.Ch <- msg
				}
			}

			fmt.Println(":::successfully disconnected!:::")
			os.Exit(1)
		case msg := <-messages:
			for cli, _ := range clients {
				if cli.IP != msg.SenderIP {
					cli.Ch <- msg
				}
			}
		case cli := <-entering:
			clients[cli] = true
		case cli := <-leaving:
			delete(clients, cli)
			close(cli.Ch)
		}
	}
}

func clientWriter(cli *Client) {
	for msg := range cli.Ch {
		if err := cli.Enc.Encode(msg); err != nil {
			//log.Print(err)
			continue
		}
	}
}

func NewClient(conn net.Conn) *Client {
	return &Client{
		Ch:  make(chan Message),
		Dec: json.NewDecoder(conn),
		Enc: json.NewEncoder(conn),
		IP:  conn.RemoteAddr().String(),
	}
}
