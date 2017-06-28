package main

import (
	"encoding/binary"
	"fmt"
	"github.com/akshay1713/goUtils"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"io/ioutil"
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
	sendingFiles   MultipleTransferFiles
	receivingFiles MultipleTransferFiles
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

func (peer *Peer) sendFile(file TransferFile) {
	log.Println("Sending file ", file.filePath)
	fileData := file.getNextBytes()
	for len(fileData) > 0 {
		fileDataMsg := getFileDataMsg(fileData, file.uniqueID, file.getFileName())
		peer.sendingFiles = peer.sendingFiles.update(file)
		peer.sendMessage(fileDataMsg)
		fileData = file.getNextBytes()
	}
	peer.sendingFiles = peer.sendingFiles.remove(file.filePath)
}

func (peer *Peer) fileDataHandler(fileDataMsg []byte) {
	var file TransferFile
	uniqueID, fileName, fileData := extractFileData(fileDataMsg)
	for i := range peer.receivingFiles {
		if peer.receivingFiles[i].uniqueID == uniqueID && peer.receivingFiles[i].getFileName() == fileName {
			file = peer.receivingFiles[i]
			break
		}
	}
	finished := file.writeBytes(fileData)
	peer.receivingFiles = peer.receivingFiles.update(file)
	if finished {
		peer.receivingFiles = peer.receivingFiles.remove(file.filePath)
	}
}

func (peer *Peer) fileReqHandler(fileReqMsg []byte) {
	uniqueID, fileName, diffType := extractFileReqMsg(fileReqMsg)
	log.Println("Diff type is ", diffType)
	filePath := peer.folderManager.getFilePath(uniqueID, fileName)
	lockFile := filePath + ".lock"
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		log.Println("Lock file for", filePath, "exists, continuing")
		return
	}
	filePtr, err := os.Open(filePath)
	goUtils.HandleErr(err, "While opening file for sending" + filePath)
	lockPtr, err := os.Create(lockFile)
	goUtils.HandleErr(err, "While creating lock file " + lockFile)
	fileStat, err := filePtr.Stat()
	goUtils.HandleErr(err, "While geting file stats")
	modTime := fileStat.ModTime().UTC().Unix()
	modTimeBytes := []byte(strconv.FormatInt(modTime, 10))
	lockPtr.Write(modTimeBytes)
	lockPtr.Close()
	fileSize := fileStat.Size()
	transferFile := TransferFile{
		filePath:        filePath,
		filePtr:         filePtr,
		fileSize:        uint64(fileSize),
		transferredSize: 0,
		uniqueID:        uniqueID,
	}
	peer.sendingFiles = append(peer.sendingFiles, transferFile)
	go peer.sendFile(transferFile)
}

func (peer *Peer) syncReqHandler(syncReqMsg []byte) {
	diffType, uniqueID, fileSizes, fileNames, md5Hashes, modTimes := extractSyncReqMsg(syncReqMsg)
	uniqueIDs := peer.folderManager.getAllUniqueIDs()
	uniqueIDstring := strconv.FormatInt(int64(uniqueID), 10)
	if goUtils.Pos(uniqueIDs, uniqueIDstring) == -1 {
		peer.cliController.print(peer.username + " wants to sync a folder with the following details\n" +
			"uniqueid - " + string(uniqueID) + "\nFiles - " + strings.Join(fileNames, ", ") + "\n" +
			"MD5 Hashes - " + strings.Join(md5Hashes, ", "))
		userResponse := peer.cliController.getInput("Do you want to accept this folder?[y/n]")
		if userResponse == "y" {
			directory := peer.cliController.getInput("Enter the directory where you want to create this folder")
			folderName := peer.cliController.getInput("Enter the name of the folder you want to create")
			peer.folderManager.addPeerFolder(directory, folderName, uniqueID, fileNames)
			for i := range fileNames {
				log.Println(modTimes[i])
				fileReqMsg := getFileReqMsg(int64(uniqueID), fileNames[i], 1)
				filePath := directory + "/" + folderName + "/" + fileNames[i]
				filePtr, err := os.OpenFile(filePath, os.O_TRUNC|os.O_WRONLY, 0755)
				goUtils.HandleErr(err, "While opening file for writing")
				transferFile := TransferFile{
					filePath:        filePath,
					transferredSize: 0,
					fileSize:        fileSizes[i],
					filePtr:         filePtr,
					uniqueID:        uniqueID,
				}
				peer.receivingFiles = append(peer.receivingFiles, transferFile)
				peer.sendMessage(fileReqMsg)
			}
		}
	} else {
		//sync existing folder here
		if diffType == 1 {
			log.Println("Received sync request for folder with details \n" +
				"uniqueid - " + string(uniqueID) + "\nFiles - " + strings.Join(fileNames, ", ") + "\n")
			syncData := peer.folderManager.updateAndGetSyncData(uniqueID)
			currentMd5Hashes := make(map[string]string)
			for i := range syncData.Files {
				currentMd5Hashes[syncData.Files[i].Name] = syncData.Files[i].Md5
			}

			changedFileNames := []string{}
			changedFileSizes := []uint64{}
			changedModTimes := []uint32{}
			for i := range fileNames {
				if md5Hashes[i] == currentMd5Hashes[fileNames[i]] {
					log.Println(fileNames[i], "has not changed, continuing")
					continue
				}
				changedFileNames = append(changedFileNames, fileNames[i])
				changedFileSizes = append(changedFileSizes, fileSizes[i])
				changedModTimes = append(changedModTimes, modTimes[i])
			}
			folderPath := peer.folderManager.backupExistingFiles(uniqueID, changedFileNames)
			for i := range changedFileNames {
				lockFile := folderPath + "/." + changedFileNames[i] + ".lock"
				if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
					log.Println("Lock file for", changedFileNames[i], "exists, continuing")
					timeBytes, err := ioutil.ReadFile(lockFile)
					goUtils.HandleErr(err, "While reading existing lock file " + lockFile)
					timeInt, err := strconv.Atoi(string(timeBytes))
					goUtils.HandleErr(err, "While converting lock file contents to int")
					currentModTime := uint32(timeInt)
					if currentModTime > changedModTimes[i] {
						peer.folderManager.restoreFile(uniqueID, changedFileNames[i])
						log.Println("This file is already being received with a higher mod time")
						continue
					} else {
						filePath := folderPath + "/" + changedFileNames[i]
						peer.receivingFiles = peer.receivingFiles.remove(filePath)
						peer.sendingFiles = peer.sendingFiles.remove(filePath)
					}
				}
				lockPtr, err := os.Create(lockFile)
				goUtils.HandleErr(err, "While creating lock file for "+changedFileNames[i])
				lockPtr.Write([]byte(strconv.FormatInt(int64(changedModTimes[i]), 10)))
				lockPtr.Close()
				fileReqMsg := getFileReqMsg(int64(uniqueID), changedFileNames[i], 1)
				filePath := folderPath + "/" + changedFileNames[i]
				filePtr, err := os.OpenFile(filePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0755)
				goUtils.HandleErr(err, "While opening file for writing")
				transferFile := TransferFile{
					filePath:        filePath,
					transferredSize: 0,
					fileSize:        changedFileSizes[i],
					filePtr:         filePtr,
					uniqueID:        uniqueID,
				}
				peer.receivingFiles = append(peer.receivingFiles, transferFile)
				peer.sendMessage(fileReqMsg)
			}
		}
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
