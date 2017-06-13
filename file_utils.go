package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func getMD5Hash(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error while getting md5 hash for file", err)
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Println("Error while getting md5 hash for file", err)
		return returnMD5String, err
	}
	//Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]
	//Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil

}

func getFileNamesInFolder(folderPath string) []string {
	files, _ := ioutil.ReadDir(folderPath)
	filesInFolder := []string{}
	for _, f := range files {
		if !f.IsDir() {
			filesInFolder = append(filesInFolder, f.Name())
		}
	}
	return filesInFolder
}
