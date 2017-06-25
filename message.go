package main

import (
	"encoding/binary"
	"github.com/akshay1713/goUtils"
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

func getFileReqMsg(uniqueID int64, fileName string, diffType byte) []byte {
	fileReqMsg := make([]byte, 5+len(fileName)+4)
	msgLen := len(fileName) + 4
	goUtils.GetBytesFromUint32(fileReqMsg[0:4], uint32(msgLen)+1)
	fileReqMsg[4] = 3
	fileReqMsg[5] = diffType
	goUtils.GetBytesFromUint32(fileReqMsg[6:10], uint32(uniqueID))
	copy(fileReqMsg[10:], fileName)
	return fileReqMsg
}

func extractFileReqMsg(fileReqMsg []byte) (uint32, string, byte) {
	diffType := fileReqMsg[1]
	uniqueID := binary.BigEndian.Uint32(fileReqMsg[2:6])
	fileName := string(fileReqMsg[6:])
	return uniqueID, fileName, diffType
}

func getFileDataMsg(fileData []byte, uniqueID uint32, fileName string) []byte {
	fileDataMsg := make([]byte, 5+len(fileData)+32+len(fileName)+1)
	msgLen := len(fileData) + 32 + len(fileName) + 1
	goUtils.GetBytesFromUint32(fileDataMsg[0:4], uint32(msgLen)+1)
	fileDataMsg[4] = 4
	fileDataMsg[5] = byte(len(fileName))
	copy(fileDataMsg[6:6+len(fileName)], fileName)
	position := 6 + len(fileName)
	goUtils.GetBytesFromUint32(fileDataMsg[position:position+32], uniqueID)
	position += 32
	copy(fileDataMsg[position:], fileData)
	return fileDataMsg
}

func extractFileData(fileDataMsg []byte) (uint32, string, []byte) {
	fileNameLen := int(fileDataMsg[1])
	fileName := string(fileDataMsg[2 : 2+fileNameLen])
	position := 2 + fileNameLen
	uniqueID := binary.BigEndian.Uint32(fileDataMsg[position : position+32])
	position += 32
	fileData := fileDataMsg[position:]
	return uniqueID, fileName, fileData
}

func getFileInfoMsg(fileLen uint64, fileName string, md5 string, uniqueID uint32) []byte {
	fileNameLen := uint8(len(fileName))
	fileMsgLen := 10 + fileNameLen + 32 + 4
	fileMsg := make([]byte, fileMsgLen+4)
	goUtils.GetBytesFromUint32(fileMsg[0:4], uint32(fileMsgLen))
	fileMsg[4] = 3
	fileMsg[5] = fileNameLen
	goUtils.GetBytesFromUint64(fileMsg[6:], fileLen)
	copy(fileMsg[14:], md5)
	goUtils.GetBytesFromUint32(fileMsg[46:50], uniqueID)
	copy(fileMsg[50:], fileName)
	return fileMsg
}

func getSyncReqMsg(uniqueID uint32, diffType byte, fileNames []string, fileSizes []uint64, md5Hashes []string, modTimes []uint32) []byte {
	totalNameLen := 0
	for i := range fileNames {
		totalNameLen += len(fileNames[i])
	}
	syncReqMsg := make([]byte, 10+totalNameLen+2+len(fileNames)+8*len(fileSizes)+32*len(md5Hashes) + 32*len(modTimes))
	msgLen := 6 + totalNameLen + 2 + len(fileNames) + 8*len(fileSizes) + 32*len(md5Hashes) + 32*len(modTimes)
	goUtils.GetBytesFromUint32(syncReqMsg[0:4], uint32(msgLen))
	syncReqMsg[4] = 2
	syncReqMsg[5] = diffType
	goUtils.GetBytesFromUint16(syncReqMsg[6:8], uint16(len(fileNames)))
	goUtils.GetBytesFromUint32(syncReqMsg[8:12], uniqueID)
	start := 12
	for i := range fileNames {
		syncReqMsg[start] = byte(len(fileNames[i]))
		start++
	}
	for i := range fileSizes {
		goUtils.GetBytesFromUint64(syncReqMsg[start:start+8], fileSizes[i])
		start += 8
	}
	for i := range md5Hashes {
		copy(syncReqMsg[start:start+32], md5Hashes[i])
		start += 32
	}
	for i := range modTimes {
		goUtils.GetBytesFromUint32(syncReqMsg[start:start+32], modTimes[i])
		start += 32
	}
	for i := range fileNames {
		copy(syncReqMsg[start:start+len(fileNames[i])], fileNames[i])
		start += len(fileNames[i])
	}
	return syncReqMsg
}

func extractSyncReqMsg(syncReqMsg []byte) (byte, uint32, []uint64, []string, []string, []uint32) {
	num_files := binary.BigEndian.Uint16(syncReqMsg[2:4])
	folderID := uint32(binary.BigEndian.Uint32(syncReqMsg[4:8]))
	start := 8
	name_lengths := []byte{}
	for ; start < int(num_files)+8; start++ {
		name_lengths = append(name_lengths, syncReqMsg[start])
	}
	fileSizes := []uint64{}
	for i := 0; i < int(num_files); i++ {
		fileSizes = append(fileSizes, binary.BigEndian.Uint64(syncReqMsg[start:start+8]))
		start += 8
	}
	md5Hashes := []string{}
	for i := 0; i < int(num_files); i++ {
		md5Hashes = append(md5Hashes, string(syncReqMsg[start:start+32]))
		start += 32
	}
	modTimes := []uint32{}
	for i := 0; i < int(num_files); i++ {
		modTimes = append(modTimes, binary.BigEndian.Uint32(syncReqMsg[start:start+32]))
	}
	fileNames := []string{}
	for i := range name_lengths {
		fileNames = append(fileNames, string(syncReqMsg[start:start+int(name_lengths[i])]))
		start += int(name_lengths[i])
	}
	return syncReqMsg[1], folderID, fileSizes, fileNames, md5Hashes, modTimes
}

func getMsgType(msg []byte) string {
	availableMsgTypes := map[byte]string{
		0: "ping",
		1: "pong",
		2: "sync_req",
		3: "file_req",
		4: "file_data",
	}
	msgType := availableMsgTypes[msg[0]]
	return msgType
}
