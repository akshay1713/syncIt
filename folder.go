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
	cliController CLIController
}

func (folder FolderManager) add(folderPath string) {
	syncFolder := folderPath + "/.syncIt"
	log.Println("Creating folder ", syncFolder)
	if _, err := os.Stat(syncFolder); !os.IsNotExist(err) {
		os.RemoveAll(syncFolder)
	}
	err := os.Mkdir(syncFolder, 0755)
	// path/to/whatever does not exist
	if err != nil {
		folder.cliController.print("Error while creating sync config directory " + string(err.Error()))
		return
	}
	configFile := syncFolder + "/.syncit.json"
	_, err = os.Create(configFile)
	goUtils.HandleErr(err, "Error while creating config file:")
	_ = addMultipleFiles(folderPath, configFile)
	folder.addToGlobal(folderPath)
}

func (folder FolderManager) addToGlobal(folderPath string){
	uniqueID := time.Now().UTC().Unix()
	absFolderPath, _ := filepath.Abs(folderPath)
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

func getGlobalConfig() string{
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
}

func (folder FolderManager) sync(folderPath string) {
	syncFolder := folderPath + "/.syncIt"
	if _, err := os.Stat(syncFolder); os.IsNotExist(err) {
		folder.cliController.print("This is an unsynced folder, adding it for syncing")
		folder.add(folderPath)
	}
	//filesInFolder := getFileNamesInFolder(folderPath)
	syncData := getSyncData(folderPath, syncFolder+"/.syncit.json")
	oldSyncData := syncData
	syncData.update(folderPath, syncFolder+"/.syncit.json")
	changedFiles := syncData.getChangedFiles(oldSyncData)
	if len(changedFiles) == 0 {
		folder.cliController.print("No files changed. Resync not required.")
		return
	}
	syncReqMessages := [][]byte{}
	for i:= range changedFiles {
		syncReqMessages[i] = getSyncReqMsg(syncData.UniqueID, 1, changedFiles[i].Name)
	}
}
