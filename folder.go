package main

import (
	"encoding/json"
	"github.com/akshay1713/goUtils"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"
)

type FolderManager struct {
	peermanager   PeerManager
	cliController *CLIController
}

func (folder FolderManager) setupFolderConfig(folderPath string) string {
	syncFolder := folderPath + "/.syncIt"
	log.Println("Creating folder ", syncFolder)
	if _, err := os.Stat(syncFolder); !os.IsNotExist(err) {
		os.RemoveAll(syncFolder)
	}
	err := os.Mkdir(syncFolder, 0755)
	// path/to/whatever does not exist
	if err != nil {
		folder.cliController.print("Error while creating sync config directory " + string(err.Error()))
		return ""
	}
	configFile := syncFolder + "/.syncIt.json"
	_, err = os.Create(configFile)
	goUtils.HandleErr(err, "Error while creating config file:")
	return configFile
}

func (folder FolderManager) add(folderPath string) {
	configFile := folder.setupFolderConfig(folderPath)
	uniqueID := folder.addNewFolderToGlobal(folderPath)
	_ = addMultipleFiles(folderPath, configFile, uniqueID)
}

func (folder FolderManager) addNewFolderToGlobal(folderPath string) uint32 {
	uniqueID := getNewUniqueID()
	absFolderPath, _ := filepath.Abs(folderPath)
	folder.addToGlobal(absFolderPath, uniqueID)
	return uint32(uniqueID)
}

func (folder FolderManager) addToGlobal(absFolderPath string, uniqueID uint32) {
	globalConfigFile := getGlobalConfigFile()
	configBytes, err := ioutil.ReadFile(globalConfigFile)
	goUtils.HandleErr(err, "While reading current config file")
	globalConfigJson := make(map[string]string)
	json.Unmarshal(configBytes, &globalConfigJson)
	log.Println("Current config is ", globalConfigJson)
	globalConfigJson[strconv.FormatInt(int64(uniqueID), 10)] = absFolderPath
	marshalledConfig, _ := json.Marshal(globalConfigJson)
	ioutil.WriteFile(globalConfigFile, marshalledConfig, 0755)
}

func (folder FolderManager) sync(folderPath string) {
	syncData := folder.updateExistingFolderConfig(folderPath)
	filesInFolder := syncData.getAllFiles()
	fileNames := []string{}
	for i := range filesInFolder {
		fileNames = append(fileNames, filesInFolder[i].Name)
	}
	fileSizes := []uint64{}
	for i := range filesInFolder {
		fileSizes = append(fileSizes, filesInFolder[i].Size)
	}
	md5Hashes := []string{}
	for i := range filesInFolder {
		md5Hashes = append(md5Hashes, filesInFolder[i].Md5)
	}
	syncReqMsg := getSyncReqMsg(syncData.UniqueID, 1, fileNames, fileSizes, md5Hashes)
	folder.peermanager.sendToAllPeers(syncReqMsg)
}

func (folder FolderManager) updateExistingFolderConfig(folderPath string) SyncData {
	syncFolder := folderPath + "/.syncIt"
	if _, err := os.Stat(syncFolder); os.IsNotExist(err) {
		folder.cliController.print("This is an unsynced folder, adding it for syncing")
		folder.add(folderPath)
	}
	syncData := getSyncData(folderPath, syncFolder+"/.syncIt.json")
	syncData.update(folderPath, syncFolder+"/.syncIt.json")
	return syncData
}

func (folder FolderManager) getAllUniqueIDs() []string {
	globalConfigJson := getGlobalConfig()
	uniqueIDs := []string{}
	for id, _ := range globalConfigJson {
		uniqueIDs = append(uniqueIDs, id)
	}
	return uniqueIDs
}

func (folder FolderManager) getAllFolders() []string {
	globalConfigJson := getGlobalConfig()
	absFolderPaths := []string{}
	for _, absFolderPath := range globalConfigJson {
		absFolderPaths = append(absFolderPaths, absFolderPath)
	}
	return absFolderPaths
}

func (folder FolderManager) getFolderPath(uniqueID uint32) string {
	uniqueIDstring := strconv.FormatInt(int64(uniqueID), 10)
	return getGlobalConfig()[uniqueIDstring]
}

func (folder FolderManager) backupExistingFiles(uniqueID uint32, fileNames []string) string {
	folderPath := folder.getFolderPath(uniqueID)
	syncFolder := folderPath + "/.syncIt"
	for i := range fileNames {
		log.Println("Moving ", fileNames[i])
		currentFilePath := folderPath + "/" + fileNames[i]
		newFilePath := syncFolder + "/" + fileNames[i] + ".bak"
		os.Rename(currentFilePath, newFilePath)
	}
	return folderPath
}

func (folder FolderManager) addPeerFolder(directory string, folderName string, uniqueID uint32, fileNames []string) {
	folderPath := directory + "/" + folderName
	err := os.Mkdir(folderPath, 0755)
	goUtils.HandleErr(err, "While creating peer folder")
	folderConfigFile := folder.setupFolderConfig(folderPath)
	absFolderPath, err := filepath.Abs(folderPath)
	goUtils.HandleErr(err, "While getting absolute folder path")
	folder.addToGlobal(absFolderPath, uniqueID)
	folder.addPeerFiles(folderPath, fileNames, folderConfigFile, uniqueID)
}

func (folder FolderManager) addPeerFiles(folderPath string, fileNames []string, configPath string, uniqueID uint32) {
	for i := range fileNames {
		log.Println("Creating file ", fileNames[i])
		_, err := os.Create(folderPath + "/" + fileNames[i])
		goUtils.HandleErr(err, "While creating file")
	}
	addMultipleFiles(folderPath, configPath, uniqueID)
}

func (folder FolderManager) updateAndGetSyncData(uniqueID uint32) SyncData {
	folderPath := folder.getFolderPath(uniqueID)
	folder.updateExistingFolderConfig(folderPath)
	configPath := folderPath + "/.syncIt/.syncIt.json"
	syncData := getSyncData(folderPath, configPath)
	return syncData
}

func (folder FolderManager) getFilePath(uniqueID uint32, fileName string) string {
	globalConfigJson := getGlobalConfig()
	uniqueIDString := strconv.FormatInt(int64(uniqueID), 10)
	folderPath := globalConfigJson[uniqueIDString]
	return folderPath + "/" + fileName
}

func getGlobalConfigFile() string {
	user, _ := user.Current()
	homeDir := user.HomeDir
	globalConfigFolder := filepath.Join(homeDir, ".syncIt")
	if _, err := os.Stat(globalConfigFolder); os.IsNotExist(err) {
		log.Println("Creating config directory", globalConfigFolder)
		err := os.Mkdir(globalConfigFolder, 0755)
		goUtils.HandleErr(err, "While creating config folder")
	}
	globalConfigFile := globalConfigFolder + "/global.json"
	if _, err := os.Stat(globalConfigFile); os.IsNotExist(err) {
		log.Println("Creating config file", globalConfigFile)
		config, _ := os.Create(globalConfigFile)
		emptyConfig, _ := json.Marshal(struct{}{})
		config.Write(emptyConfig)
		config.Close()
	}
	return globalConfigFile
}

func getGlobalConfig() map[string]string {
	globalConfigFile := getGlobalConfigFile()
	configBytes, err := ioutil.ReadFile(globalConfigFile)
	goUtils.HandleErr(err, "While reading global config file")
	globalConfigJson := make(map[string]string)
	json.Unmarshal(configBytes, &globalConfigJson)
	return globalConfigJson
}

func getNewUniqueID() uint32 {
	return uint32(time.Now().UTC().Unix())
}
