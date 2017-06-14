package main

import (
	"github.com/akshay1713/goUtils"
	"log"
)

func getPingMsg() []byte {
	pingMsg := make([]byte, 5)
	copy(pingMsg[0:4], []byte{0, 0, 0, 1})
	copy(pingMsg[4:5], []byte{0})
	return pingMsg
}

func getPongMsg() []byte {
	pingMsg := make([]byte, 5)
	copy(pingMsg[0:4], []byte{0, 0, 0, 1})
	copy(pingMsg[4:5], []byte{1})
	return pingMsg
}

func getFileDataMsg(fileData []byte, uniqueID uint32) []byte {
	fileDataMsg := make([]byte, 5+len(fileData)+32)
	msgLen := len(fileData) + 32
	goUtils.GetBytesFromUint32(fileDataMsg[0:4], uint32(msgLen)+1)
	fileDataMsg[4] = 5
	goUtils.GetBytesFromUint32(fileDataMsg[5:37], uniqueID)
	copy(fileDataMsg[37:], fileData)
	return fileDataMsg
}

func getSyncReqMsg(uniqueID int64, diffType byte, fileNames []string) []byte{
	log.Println("Sending filenames ", fileNames)
	totalNameLen := 0
	for i := range fileNames {
		totalNameLen += len(fileNames[i])
	}
	syncReqMsg := make([]byte, 10 + totalNameLen + 2 + len(fileNames))
	msgLen := 6 + totalNameLen + 2 + len(fileNames)
	goUtils.GetBytesFromUint32(syncReqMsg[0:4], uint32(msgLen))
	syncReqMsg[4] = 2
	syncReqMsg[5] = diffType
	goUtils.GetBytesFromUint16(syncReqMsg[6:8], uint16(len(fileNames)))
	goUtils.GetBytesFromUint32(syncReqMsg[8:12], uint32(uniqueID))
	start := 12
	for i := range fileNames {
		syncReqMsg[start] = byte(len(fileNames[i]))
		start++
	}
	for i := range fileNames {
		syncReqMsg = append(syncReqMsg, fileNames[i]...)
		copy(syncReqMsg[start:start+len(fileNames[i])], fileNames[i])
		start += len(fileNames[i])
	}
	log.Println("Sending message ", syncReqMsg)
	return syncReqMsg
}

func getMsgType(msg []byte) string {
	availableMsgTypes := map[byte]string{
		0: "ping",
		1: "pong",
		2: "sync_req",
		3: "file_info",
		4: "file_accept",
		5: "file_data",
	}
	msgType := availableMsgTypes[msg[0]]
	return msgType
}
