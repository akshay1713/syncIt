package main

import (
	"encoding/json"
	"github.com/akshay1713/goUtils"
	"io/ioutil"
	"os"
	"time"
)

type TransferFile struct {
	filePath           string
	fileSize           uint64
	transferredSize    uint64
	md5                string
	filePtr            *os.File
	uniqueID           uint32
}

func newTransferFile(filePath string, fileSize uint64) TransferFile{
	transferFile := TransferFile{}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		transferFile.filePtr, _ = os.OpenFile(filePath, os.O_RDWR | os.O_CREATE, 0755)
	} else {
		transferFile.filePtr, _ = os.Open(filePath)
	}
	transferFile.transferredSize = fileSize
	return transferFile
}

type SyncFile struct {
	Md5  string `json:"md5"`
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

type SyncData struct {
	UniqueID   int64      `json:"unique_id"`
	Files      []SyncFile `json:"files"`
	Synced     bool       `json:"synced"`
	LastSynced int64      `json:"last_synced"`
}

func addMultipleFiles(folderPath string, configPath string) []SyncFile {
	files := []SyncFile{}
	fileNames := getFileNamesInFolder(folderPath)
	for i := range fileNames {
		md5, _ := getMD5Hash(folderPath + "/" + fileNames[i])
		filePtr, _ := os.Open(folderPath + "/" + fileNames[i])
		fileStat, _ := filePtr.Stat()
		fileSize := uint64(fileStat.Size())
		files = append(files, SyncFile{Md5: md5, Name: fileNames[i], Size: fileSize})
	}
	syncData := SyncData{Files: files, UniqueID: time.Now().UTC().Unix()}
	syncDataBytes, _ := json.Marshal(syncData)
	ioutil.WriteFile(configPath, syncDataBytes, 0755)
	return files
}

func (syncData *SyncData) update(folderPath string, configPath string) {
	syncData.Files = addMultipleFiles(folderPath, configPath)
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
