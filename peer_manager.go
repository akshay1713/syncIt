package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

type PeerManager struct {
	closeChan      chan Peer
	connectedPeers map[string]*Peer
}

func (peerManager PeerManager) IsConnected(IP string) bool {
	return false
}

func (peerManager PeerManager) GetAllIPs() []string {
	return []string{}
}

func (peerManager PeerManager) addNewPeer(conn *net.TCPConn, currentTimestamp uint32, initiated bool, username string, cliController *CLIController) Peer {
	if initiated {
		conn.Write([]byte{1})
		currentTimestampBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(currentTimestampBytes, currentTimestamp)
		conn.Write(currentTimestampBytes)
	}
	usernameBytes := make([]byte, len(username)+2)
	binary.BigEndian.PutUint16(usernameBytes[0:2], uint16(len(username)))
	copy(usernameBytes[2:], username)
	conn.Write(usernameBytes)
	peerUsernameLenBytes := make([]byte, 2)
	conn.Read(peerUsernameLenBytes)
	peerUsernameLen := binary.BigEndian.Uint16(peerUsernameLenBytes)
	peerUsername := make([]byte, peerUsernameLen)
	conn.Read(peerUsername)
	newPeer := Peer{Conn: conn, closeChan: peerManager.closeChan, connected: true, username: string(peerUsername), cliController: cliController}
	fmt.Println("Connected to ", string(peerUsername))
	peerAddress := conn.RemoteAddr().String()
	peerIP := strings.Split(peerAddress, ":")[0]
	peerManager.connectedPeers[peerIP] = &newPeer
	newPeer.initPeer()
	return newPeer
}

func (peerManager *PeerManager) compareTimestampAndUpdate(conn *net.TCPConn, newTimestamp uint32, IP string) {
	peer, exists := peerManager.connectedPeers[IP]
	if !exists {
		fmt.Println("Peer to update not found", IP)
		return
	}
	if peer.connectedAt < newTimestamp {
		fmt.Println("current timestamp is older, not updating")
		return
	}
	fmt.Println("Updating existing peer")
	peer.disConnect()
	peer.Conn = conn
	peer.connectedAt = newTimestamp
	go peer.listenForMessages()
}

func (peerManager PeerManager) sendToAllPeers(msg []byte) {
	for _, peer := range peerManager.connectedPeers {
		peer.sendMessage(msg)
	}
}
