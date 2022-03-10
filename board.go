package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"inet.af/netaddr"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
	"github.com/netsec-ethz/scion-apps/pkg/quicutil"
)

type Room uint8

const (
	Lobby Room = iota
	Chat
	Wordle
)

type Client struct {
	id            uint32
	room          Room
	name          string
	conn          net.Conn
	wordleGuesses chan string
}

type Message struct {
	content  string
	senderId uint32
}

func main() {
	clients := make(map[uint32]*Client)

	clientsNew := make(chan Client)
	clientsDone := make(chan uint32)

	clientsInput := make(chan Message)

	wordleInfo := make(chan Message)

	go listen(clientsNew, clientsDone, clientsInput)

	for {
		select {
		case client := <-clientsNew:
			clients[client.id] = &client
			lobbyWelcome(&client)

		case id := <-clientsDone:
			chatBroadcast(clients, fmt.Sprintf("\x1b[3m%s disconnected!\x1b[0m\n", clients[id].name))
			delete(clients, id)

		case msg := <-clientsInput:
			go handleMessage(clients, msg, wordleInfo)

		case msg := <-wordleInfo:
			if msg.content == "DONE" {
				clients[msg.senderId].room = Lobby
				lobbyWelcome(clients[msg.senderId])
			} else {
				io.WriteString(clients[msg.senderId].conn, msg.content)
			}
		}
	}
}

func handleMessage(clients map[uint32]*Client, msg Message, wordleInfo chan<- Message) {
	switch clients[msg.senderId].room {
	case Lobby:
		switch msg.content {
		case "chat":
			clients[msg.senderId].room = Chat
			io.WriteString(clients[msg.senderId].conn, "\x1b[2J\x1b[50B")
			chatBroadcast(clients, fmt.Sprintf("\x1b[3m%s has joined the chat!\x1b[0m\n", clients[msg.senderId].name))

		case "wordle":
			clients[msg.senderId].room = Wordle
			go wordle(clients[msg.senderId].wordleGuesses, clients[msg.senderId], wordleInfo)

		default:
			lobbyWelcome(clients[msg.senderId])
		}

	case Chat:
		if msg.content == "/lobby" {
			clients[msg.senderId].room = Lobby
			chatBroadcast(clients, fmt.Sprintf("\x1b[3m%s has left the chat!\x1b[0m\n", clients[msg.senderId].name))
			lobbyWelcome(clients[msg.senderId])
		} else {
			chatBroadcast(clients, fmt.Sprintf("%s: %s\n", clients[msg.senderId].name, msg.content))
		}

	case Wordle:
		clients[msg.senderId].wordleGuesses <- msg.content
	}
}

func listen(clientsNew chan<- Client, clientsDone chan<- uint32, clientsInput chan<- Message) {
	quicListener, err := pan.ListenQUIC(
		context.Background(),
		netaddr.IPPortFrom(netaddr.IP{}, 1337),
		nil,
		&tls.Config{
			Certificates: quicutil.MustGenerateSelfSignedCert(),
			NextProtos:   []string{quicutil.SingleStreamProto},
		},
		nil,
	)
	if err != nil {
		log.Printf("Failed to listen (%v)", err)
	}
	listener := quicutil.SingleStreamListener{Listener: quicListener}

	log.Printf("Starting to wait for connections")
	for i := uint32(0); ; i++ {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept session: %v", err)
			continue
		}
		log.Printf("New connection")

		go handleConnection(i, conn, clientsNew, clientsDone, clientsInput)
	}
}

func handleConnection(clientId uint32, conn net.Conn, clientsNew chan<- Client,
	clientsDone chan<- uint32, clientsInput chan<- Message) {

	reader := bufio.NewReader(conn)

	io.WriteString(conn, "\x1b[2J\x1b[50BWelcome to the SCIONLab board!\n")
	name := "\n"
	var err error
	for name == "\n" {
		io.WriteString(conn, "Please Enter your name: ")
		name, err = reader.ReadString('\n')
		if err != nil {
			log.Printf("Could not read name: %v", err)
			conn.Close()
			return
		}
	}
	name = strings.TrimRight(name, "\n")

	clientsNew <- Client{
		id:            clientId,
		name:          name,
		conn:          conn,
		room:          Lobby,
		wordleGuesses: make(chan string),
	}

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			break
		}
		// Clear the typed line
		io.WriteString(conn, ("\x1b[1A\x1b[0J"))

		clientsInput <- Message{strings.TrimRight(message, "\n"), clientId}
	}
	clientsDone <- clientId
}

func chatBroadcast(clients map[uint32]*Client, str string) {
	for _, c := range clients {
		if c.room == Chat {
			io.WriteString(c.conn, str)
		}
	}
}

func lobbyWelcome(client *Client) {
	io.WriteString(client.conn, "\nYou are in the Lobby. Type\n")
	io.WriteString(client.conn, "* \x1b[3mchat\x1b[0m to enter the Chat\n")
	io.WriteString(client.conn, "* \x1b[3mwordle\x1b[0m to play Wordle\n")
	io.WriteString(client.conn, "You can type \x1b[3m/lobby\x1b[0m anytime to come back.\n ")
}
