package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/akshay1713/LANPeerDiscovery"
	"github.com/akshay1713/goUtils"
	"io"
	"strings"
	"time"
)

func main() {
	username := getUserName()
	if username == "" {
		fmt.Println("Please specify a username using the -u flag")
		return
	}
	fmt.Println("Looking for peers")
	connectedPeers := make(map[string]*Peer)
	closeChan := make(chan Peer)
	peerManager := PeerManager{closeChan: closeChan, connectedPeers: connectedPeers}
	inputChan := make(chan string)
	cliController := CLIController{inputChan: inputChan}
	go initDiscovery(peerManager, username, &cliController)
	folder := FolderManager{cliController: &cliController, peermanager: peerManager}
	cliController.startCli(folder)
}

func getUserName() string {
	var usernamePtr *string
	usernamePtr = flag.String("u", "", "Desired username")
	flag.Parse()
	return *usernamePtr
}

func initDiscovery(peerManager PeerManager, username string, cliController *CLIController) {
	candidatePorts := []string{"8011", "8012"}
	connectionsChan := LANPeerDiscovery.GetConnectionsChan(candidatePorts, peerManager, "syncIt")
	for connAndType := range connectionsChan {
		switch connAndType.Type {
		case "sender":
			currentTimestamp := uint32(time.Now().UTC().Unix())
			peerManager.addNewPeer(connAndType.Connection, currentTimestamp, true, username, cliController)
		case "receiver":
			recvdTimestampBytes := make([]byte, 4)
			_, err := io.ReadFull(connAndType.Connection, recvdTimestampBytes)
			goUtils.HandleErr(err, "While getting timestamp")
			recvdTimestamp := binary.BigEndian.Uint32(recvdTimestampBytes)
			peerManager.addNewPeer(connAndType.Connection, recvdTimestamp, false, username, cliController)
		case "duplicate_receiver":
			recvdTimestampBytes := make([]byte, 4)
			_, err := io.ReadFull(connAndType.Connection, recvdTimestampBytes)
			goUtils.HandleErr(err, "While getting timestamp")
			recvdTimestamp := binary.BigEndian.Uint32(recvdTimestampBytes)
			senderIPString := strings.Split(connAndType.Connection.RemoteAddr().String(), ":")[0]
			peerManager.compareTimestampAndUpdate(connAndType.Connection, recvdTimestamp, senderIPString)
		}
	}
}
