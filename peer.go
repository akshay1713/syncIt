package main

import (
	"encoding/binary"
	"fmt"
	"github.com/akshay1713/goUtils"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
	"os"
)

//Peer contains the following data associated with a connected peer-
//Conn - The TCP connection with that peer
type Peer struct {
	Conn           *net.TCPConn
	closeChan      chan Peer
	connectedAt    uint32
	connected      bool
	username       string
	msgChan        chan []byte
	stopMsgChan    chan bool
	sendMutex      sync.Mutex
	cliController  *CLIController
	folderManager  FolderManager
	sendingFiles   []TransferFile
	receivingFiles []TransferFile
}

func (peer *Peer) initPeer() {
	peer.sendingFiles = []TransferFile{}
	peer.receivingFiles = []TransferFile{}
	peer.createMsgChan()
	go peer.listenForMessages()
	peer.setPing()
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
		case "sync_req":
			peer.syncReqHandler(msg)
		case "file_req":
			peer.fileReqHandler(msg)
		case "file_data":
			peer.fileDataHandler(msg)
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

func (peer *Peer) sendFile(file TransferFile){
	log.Println("Sending file ", file)
	fileData := file.getNextBytes()
	for len(fileData) > 0 {
		fileDataMsg := getFileDataMsg(fileData, file.uniqueID, file.getFileName())
		peer.updateFile(file, true)
		peer.sendMessage(fileDataMsg)
		fileData = file.getNextBytes()
	}
}

func (peer *Peer) fileDataHandler(fileDataMsg []byte){
	var file TransferFile
	uniqueID, fileName, fileData := extractFileData(fileDataMsg)
	for i := range peer.receivingFiles {
		if peer.receivingFiles[i].uniqueID == uniqueID && peer.receivingFiles[i].getFileName() == fileName {
			file = peer.receivingFiles[i]
			break
		}
	}
	file.writeBytes(fileData)
	peer.updateFile(file, false)
}

func (peer *Peer) updateFile(file TransferFile, updateSendingFiles bool) {
	if updateSendingFiles {
		for i := range peer.sendingFiles {
			if peer.sendingFiles[i].uniqueID == file.uniqueID && peer.sendingFiles[i].filePath == file.filePath {
				peer.sendingFiles[i] = file
				return
			}
		}
	} else {
		for i := range peer.receivingFiles {
			if peer.receivingFiles[i].uniqueID == file.uniqueID && peer.receivingFiles[i].filePath == file.filePath {
				peer.receivingFiles[i] = file
				return
			}
		}
	}
}

func (peer *Peer) fileReqHandler(fileReqMsg []byte) {
	uniqueID := binary.BigEndian.Uint32(fileReqMsg[1:5])
	fileName := string(fileReqMsg[5:])
	filePath := peer.folderManager.getFilePath(uniqueID, fileName)
	filePtr, err := os.Open(filePath)
	goUtils.HandleErr(err, "While opening file for reading " + filePath)
	fileStat, err := filePtr.Stat()
	goUtils.HandleErr(err, "While geting file stats")
	fileSize := fileStat.Size()
	transferFile := TransferFile{filePath: filePath, filePtr: filePtr, fileSize: uint64(fileSize), transferredSize: 0, uniqueID: uniqueID}
	peer.sendingFiles = append(peer.sendingFiles, transferFile)
	go peer.sendFile(transferFile)
}


func (peer *Peer) syncReqHandler(syncReqMsg []byte) {
	num_files := binary.BigEndian.Uint16(syncReqMsg[2:4])
	folderID := uint32(binary.BigEndian.Uint32(syncReqMsg[4:8]))
	start := 8
	name_lengths := []byte{}
	for ; start < int(num_files)+8; start++ {
		name_lengths = append(name_lengths, syncReqMsg[start])
	}
	fileSizes := []uint64{}
	for i := 0; i < int(num_files) ; i++ {
		fileSizes = append(fileSizes, binary.BigEndian.Uint64(syncReqMsg[start:start+8]))
		start += 8
	}
	log.Println("File sizes are ", fileSizes)
	fileNames := []string{}
	for i := range name_lengths {
		fileNames = append(fileNames, string(syncReqMsg[start:start+int(name_lengths[i])]))
		start += int(name_lengths[i])
	}
	uniqueIDs := peer.folderManager.getAllUniqueIDs()
	if goUtils.Pos(uniqueIDs, string(folderID)) == -1 {
		peer.cliController.print(peer.username + " wants to sync a folder with the following details\n" +
			"uniqueid - " + string(folderID) + "\nFiles - " + strings.Join(fileNames, ", ") + "\n")
		userResponse := peer.cliController.getInput("Do you want to accept this folder?[y/n]")
		if userResponse == "y" {
			directory := peer.cliController.getInput("Enter the directory where you want to create this folder")
			folderName := peer.cliController.getInput("Enter the name of the folder you want to create")
			peer.folderManager.addPeerFolder(directory, folderName, int64(folderID), fileNames)
			for i := range fileNames {
				fileReqMsg := getFileReqMsg(int64(folderID), fileNames[i])
				filePath := directory + "/" + folderName + "/" + fileNames[i]
				filePtr, err := os.OpenFile(filePath, os.O_TRUNC | os.O_WRONLY, 0755)
				goUtils.HandleErr(err, "While opening file for writing")
				transferFile := TransferFile{filePath: filePath, transferredSize: 0, fileSize: fileSizes[i], filePtr: filePtr, uniqueID:folderID}
				peer.receivingFiles = append(peer.receivingFiles, transferFile)
				peer.sendMessage(fileReqMsg)
			}
		}
	} else {
		//sync existing folder here
	}
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
