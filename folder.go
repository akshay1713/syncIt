package main

import (
	"os"
	"log"
	"github.com/akshay1713/goUtils"
	"io/ioutil"
)

type FolderManager struct {
	peermanager PeerManager
	cliController CLIController
}

func (folder FolderManager) add(folderPath string){
	syncFolder := folderPath + "/.syncIt"
	log.Println("Creating folder ", syncFolder)
	if _, err := os.Stat(syncFolder); !os.IsNotExist(err) {
		os.RemoveAll(syncFolder)
	}
	err := os.Mkdir(syncFolder, 0755)
	// path/to/whatever does not exist
	if err != nil {
		folder.cliController.print("Error while creating sync config directory "+string(err.Error()))
		return
	}
	configFile := syncFolder + "/.syncit.json"
	_, err = os.Create(configFile)
	goUtils.HandleErr(err, "Error while creating config file:")
	files, _ := ioutil.ReadDir(folderPath)
	filesInFolder := []string{}
	for _, f := range files {
		if !f.IsDir() {
			filesInFolder = append(filesInFolder, f.Name())
		}
	}
	_  = addMultipleFiles(filesInFolder, folderPath, configFile)
}
