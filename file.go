package main

import (
	"encoding/json"
	"fmt"
	"github.com/akshay1713/goUtils"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type TransferFile struct {
	filePath        string
	fileSize        uint64
	transferredSize uint64
	md5             string
	filePtr         *os.File
	uniqueID        uint32
	modTime         uint32
}

func (file *TransferFile) getNextBytes() []byte {
	remainingSize := int(file.fileSize - file.transferredSize)
	if remainingSize == 0 {
		return []byte{}
	}
	//Transfer in chunks of 4096 bytes
	bytesToTransfer := 4096
	if remainingSize < 4096 {
		bytesToTransfer = remainingSize
		fmt.Println("Finished sending file", file.filePath)
	}
	nextBytes := make([]byte, int(bytesToTransfer))
	file.transferredSize += uint64(bytesToTransfer)
	file.filePtr.Read(nextBytes)
	return nextBytes
}

func (file TransferFile) getFileName() string {
	return filepath.Base(file.filePath)
}

func (file *TransferFile) writeBytes(fileData []byte)  bool {
	_, err := file.filePtr.Write(fileData)
	goUtils.HandleErr(err, "While writing to file")
	file.transferredSize += uint64(len(fileData))
	if file.transferredSize == file.fileSize {
		file.filePtr.Close()
		fmt.Println("Finished receiving file", file.getFileName())
		return true
	}
	return false
}

type MultipleTransferFiles []TransferFile

func (multipleFiles MultipleTransferFiles) remove(filePath string) MultipleTransferFiles{
	for i := range multipleFiles {
		if multipleFiles[i].filePath == filePath {
			multipleFiles[i].filePtr.Close()
			multipleFiles = append(multipleFiles[:i], multipleFiles[i+1:]...)
			lockFile := filePath + ".lock"
			os.Remove(lockFile)
			return multipleFiles
		}
	}
	//No match found, return as is
	return multipleFiles
}

func (multipleFiles MultipleTransferFiles) update(transferFile TransferFile) MultipleTransferFiles{
	for i := range multipleFiles {
		if multipleFiles[i].filePath == transferFile.filePath {
			multipleFiles[i] = transferFile
			return multipleFiles
		}
	}
	//No match found, return as is
	return multipleFiles
}

func newTransferFile(filePath string, fileSize uint64) TransferFile {
	transferFile := TransferFile{}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		transferFile.filePtr, _ = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	} else {
		transferFile.filePtr, _ = os.Open(filePath)
	}
	transferFile.transferredSize = fileSize
	return transferFile
}

type SyncFile struct {
	Md5         string   `json:"md5"`
	Name        string   `json:"name"`
	Size        uint64   `json:"size"`
	PieceHashes []string `json:"piece_hashes"`
	PieceCount  uint32   `json:"piece_count"`
	ModTime     uint32   `json:"mod_time"`
}

type SyncData struct {
	UniqueID   uint32     `json:"unique_id"`
	Files      []SyncFile `json:"files"`
	Synced     bool       `json:"synced"`
	LastSynced int64      `json:"last_synced"`
}

func (syncData *SyncData) update(folderPath string, configPath string) {
	syncData.Files = addMultipleFiles(folderPath, configPath, syncData.UniqueID)
	syncData.Synced = true
	syncData.LastSynced = time.Now().UTC().Unix()
}

func (syncData SyncData) getAllFiles() []SyncFile {
	return syncData.Files
}

func (syncData SyncData) getChangedFiles(oldSyncData SyncData) []SyncFile {
	//Handle new files added here as well
	files := syncData.Files
	oldFiles := oldSyncData.Files
	changedFiles := []SyncFile{}
	for i := range files {
		if files[i].Md5 != oldFiles[i].Md5 {
			changedFiles = append(changedFiles, files[i])
		}
	}
	return changedFiles
}

func getSyncData(folderPath string, configPath string) SyncData {
	syncData := SyncData{}
	syncDataBytes, err := ioutil.ReadFile(configPath)
	goUtils.HandleErr(err, "Error while reading sync config file")
	json.Unmarshal(syncDataBytes, &syncData)
	return syncData
}

func addMultipleFiles(folderPath string, configPath string, uniqueID uint32) []SyncFile {
	files := []SyncFile{}
	fileNames := getFileNamesInFolder(folderPath)
	for i := range fileNames {
		md5, _ := getMD5Hash(folderPath + "/" + fileNames[i])
		filePtr, _ := os.Open(folderPath + "/" + fileNames[i])
		fileStat, _ := filePtr.Stat()
		fileSize := uint64(fileStat.Size())
		modTime := uint32(fileStat.ModTime().UTC().Unix())
		pieceHashes := []string{}
		pieceCount := 0
		for readSize := uint64(0); readSize < fileSize; {
			remaining := fileSize - readSize
			if remaining > 524288 {
				remaining = 524288
			}
			nextBytes := make([]byte, remaining)
			filePtr.Read(nextBytes)
			sha1Bytes := getSha1(nextBytes)
			pieceHashes = append(pieceHashes, string(sha1Bytes))
			pieceCount++
			readSize += remaining
		}
		files = append(files, SyncFile{
			Md5:         md5,
			Name:        fileNames[i],
			Size:        fileSize,
			PieceCount:  uint32(pieceCount),
			PieceHashes: pieceHashes,
			ModTime:     modTime,
		})

	}
	syncData := SyncData{Files: files, UniqueID: uniqueID}
	syncDataBytes, _ := json.Marshal(syncData)
	ioutil.WriteFile(configPath, syncDataBytes, 0755)
	return files
}
