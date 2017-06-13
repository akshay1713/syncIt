package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

//Peer contains the following data associated with a connected peer-
//Conn - The TCP connection with that peer
type Peer struct {
	Conn        *net.TCPConn
	closeChan   chan Peer
	connectedAt uint32
	connected   bool
	username    string
	msgChan     chan []byte
	stopMsgChan chan bool
	sendMutex   sync.Mutex
}

func (peer *Peer) initPeer() {
	peer.createMsgChan()
	go peer.listenForMessages()
	peer.setPing()
	//peer.sendingFiles = []File{}
	//peer.receivingFiles = []File{}
}

//sendMessage is the route through which all messages are sent to a peer.
//Uses a mutex(not strictly necessary)
func (peer *Peer) sendMessage(msg []byte) error {
	peer.sendMutex.Lock()
	_, err := peer.Conn.Write(msg)
	peer.sendMutex.Unlock()
	return err
}

func (peer Peer) setPing() {
	// Do NOT forget to increase this time later
	time.AfterFunc(2*time.Second, peer.sendPing)
}

func (peer Peer) sendPing() {
	if !peer.connected {
		fmt.Println("Stopping ping")
		return
	}
	time.AfterFunc(2*time.Second, peer.sendPing)
	pingMessage := getPingMsg()
	peer.sendMessage(pingMessage)
}

func (peer *Peer) listenForMessages() {
	for {
		msg := <-peer.msgChan
		if len(msg) == 0 {
			return
		}
		msgType := getMsgType(msg)
		switch msgType {
		case "ping":
			peer.pingHandler()
		}

	}
}

//createMsgChan creates a chan into which all the messages sent by a peer will be sent
func (peer *Peer) createMsgChan() {
	msgChan := make(chan []byte)
	peer.stopMsgChan = make(chan bool)
	go func() {
		for {
			select {
			case <-peer.stopMsgChan:
				fmt.Println("Stopping poll func")
				return
			default:
				msg, err := peer.getNextMessage()
				if len(msg) == 0 || err != nil {
					peer.disConnect()
					peer.stopMsgChan <- true
					return
				}
				msgChan <- msg
			}
		}
	}()
	peer.msgChan = msgChan
}

func (peer Peer) stopMsgLoop() {
	peer.stopMsgChan <- true
}

//getNextMessage gets the next message from a connected peer. Each message is preceded by 4 bytes containing the length
//of the actual message. The first byte of the actual message identifies the type of the message
func (peer Peer) getNextMessage() ([]byte, error) {
	msgLength := 4
	lengthMsg := make([]byte, msgLength)
	_, err := io.ReadFull(peer.Conn, lengthMsg)
	payloadLength := binary.BigEndian.Uint32(lengthMsg)
	msg := make([]byte, payloadLength)
	_, err = io.ReadFull(peer.Conn, msg)
	return msg, err
}

func (peer Peer) pingHandler() {
	//fmt.Println("Ping received")
}

func (peer Peer) sendPong() {
	pongMessage := getPongMsg()
	peer.Conn.Write(pongMessage)
}

func (peer *Peer) disConnect() {
	fmt.Println(peer.username, " disconnected")
	peer.Conn.Close()
	peer.connected = false
	peer.closeChan <- *peer
	close(peer.msgChan)
}

func (peer Peer) getIPWithPort() string {
	return peer.Conn.RemoteAddr().String()
}

func (peer Peer) getIPWithoutPort() string {
	return strings.Split(peer.Conn.RemoteAddr().String(), ":")[0]
}
