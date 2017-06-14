package main

import (
	"github.com/akshay1713/goUtils"
	"log"
	"os"
	"time"
	"path/filepath"
	"encoding/json"
	"io/ioutil"
	"os/user"
	"strconv"
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
	configFile := syncFolder + "/.syncit.json"
	_, err = os.Create(configFile)
	goUtils.HandleErr(err, "Error while creating config file:")
	return configFile
}

func (folder FolderManager) add(folderPath string) {
	configFile := folder.setupFolderConfig(folderPath)
	_ = addMultipleFiles(folderPath, configFile)
	folder.addNewFolderToGlobal(folderPath)
}

func (folder FolderManager) addNewFolderToGlobal(folderPath string){
	uniqueID := time.Now().UTC().Unix()
	absFolderPath, _ := filepath.Abs(folderPath)
	folder.addToGlobal(absFolderPath, uniqueID)
}

func (folder FolderManager) addToGlobal(absFolderPath string, uniqueID int64) {
	globalConfigFile := getGlobalConfig()
	configBytes, err := ioutil.ReadFile(globalConfigFile)
	goUtils.HandleErr(err, "While reading current config file")
	globalConfigJson := make(map[string]string)
	json.Unmarshal(configBytes, &globalConfigJson)
	log.Println("Current config is ", globalConfigJson)
	globalConfigJson[absFolderPath] = strconv.FormatInt(uniqueID, 10)
	marshalledConfig, _ := json.Marshal(globalConfigJson)
	ioutil.WriteFile(globalConfigFile, marshalledConfig, 0755)
}

func getGlobalConfig() string {
	user, _ := user.Current()
	homeDir := user.HomeDir
	globalConfigFolder := filepath.Join(homeDir, ".syncit")
	if _, err := os.Stat(globalConfigFolder); os.IsNotExist(err) {
		log.Println("Creating config directory", globalConfigFolder)
		err := os.Mkdir(globalConfigFolder, 0755)
		goUtils.HandleErr(err, "While creating config folder")
	}
	globalConfigFile := globalConfigFolder + "/global.json"
	if _, err := os.Stat(globalConfigFile); os.IsNotExist(err) {
		log.Println("Creating config file", globalConfigFile)
		config,_ := os.Create(globalConfigFile)
		emptyConfig, _ := json.Marshal(struct {}{})
		config.Write(emptyConfig)
		config.Close()
	}
	return globalConfigFile
}

func (folder FolderManager) sync(folderPath string) {
	syncData := folder.updateExistingFolderConfig(folderPath)
	changedFiles := syncData.getAllFiles()
	if len(changedFiles) == 0 {
		folder.cliController.print("No files changed. Resync not required.")
		return
	}
	fileNames := []string{}
	for i := range changedFiles {
		fileNames = append(fileNames, changedFiles[i].Name)
	}
	syncReqMsg := getSyncReqMsg(syncData.UniqueID, 1, fileNames)
	folder.peermanager.sendToAllPeers(syncReqMsg)
}

func (folder FolderManager) updateExistingFolderConfig(folderPath string) SyncData {
	syncFolder := folderPath + "/.syncIt"
	if _, err := os.Stat(syncFolder); os.IsNotExist(err) {
		folder.cliController.print("This is an unsynced folder, adding it for syncing")
		folder.add(folderPath)
	}
	//filesInFolder := getFileNamesInFolder(folderPath)
	syncData := getSyncData(folderPath, syncFolder+"/.syncit.json")
	syncData.update(folderPath, syncFolder+"/.syncit.json")
	return syncData
}

func (folder FolderManager) getAllUniqueIDs() []string {
	globalConfigFile := getGlobalConfig()
	configBytes, err := ioutil.ReadFile(globalConfigFile)
	goUtils.HandleErr(err, "While reading global config file")
	globalConfigJson := make(map[string]string)
	json.Unmarshal(configBytes, globalConfigJson)
	uniqueIDs := []string{}
	for id, _ := range globalConfigJson {
		uniqueIDs = append(uniqueIDs, id)
	}
	return uniqueIDs
}

func (folder FolderManager) addPeerFolder(directory string, folderName string, uniqueID int64, fileNames []string) {
	folderPath := directory + "/" + folderName
	err := os.Mkdir(folderPath, 0755)
	goUtils.HandleErr(err, "While creating peer folder")
	folder.setupFolderConfig(folderPath)
	absPath, err := filepath.Abs(folderPath)
	goUtils.HandleErr(err, "While getting absolute folder path")
	folder.addToGlobal(absPath, uniqueID)
	folder.addPeerFiles(folderPath, fileNames)
}

func (folder FolderManager) addPeerFiles(folderPath string, fileNames []string) {
	for i := range fileNames {
		log.Println("Creating file ", fileNames[i])
		_, err := os.Create(folderPath + "/" + fileNames[i])
		goUtils.HandleErr(err, "While creating file")
	}
	folder.updateExistingFolderConfig(folderPath)
}
