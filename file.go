package main

import (
	"encoding/json"
	"github.com/akshay1713/goUtils"
	"io/ioutil"
	"time"
)

type File struct {
	Md5      string `json:"md5"`
	Name     string `json:"name"`
}

type SyncData struct {
	UniqueID int64  `json:"unique_id"`
	Files      []File `json:"files"`
	Synced     bool   `json:"synced"`
	LastSynced int64  `json:"last_synced"`
}

func addMultipleFiles(folderPath string, configPath string) []File {
	files := []File{}
	fileNames := getFileNamesInFolder(folderPath)
	for i := range fileNames {
		md5, _ := getMD5Hash(folderPath + "/" + fileNames[i])
		files = append(files, File{Md5: md5, Name: fileNames[i]})
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

func (syncData SyncData) getChangedFiles(oldSyncData SyncData) []File{
	//Handle new files added here as well
	files := syncData.Files
	oldFiles := oldSyncData.Files
	changedFiles := []File{}
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
