package main

import (
	"os"
	"crypto/md5"
	"io"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
)

type File struct {
	Md5 string `json:"md5"`
	Name string `json:"name"`
}

type SyncData struct{
	Files []File `json:"files"`
}

func addMultipleFiles(fileNames []string, folderPath string, configPath string) []File{
	files := []File{}
	for i := range fileNames{
		md5, _ := getMD5Hash(folderPath + "/" + fileNames[i])
		log.Println("md5 is ", md5)
		files = append(files, File{Md5: md5, Name:fileNames[i]})
	}
	syncData := SyncData{Files: files}
	jsonBytes, _ := json.Marshal(syncData)
	ioutil.WriteFile(configPath, jsonBytes, 0755)
	return files
}

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
