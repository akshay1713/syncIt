package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

type CLIController struct {
	userIO    sync.Mutex
	ioWait    bool
	inputChan chan string
}

func (cliController *CLIController) getInput(prompt string) string {
	cliController.lock()
	cliController.ioWait = true
	fmt.Println(prompt)
	text := <-cliController.inputChan
	cliController.ioWait = false
	cliController.unlock()
	return text
}

func (cliController *CLIController) getCommandInput(prompt string) string {
	fmt.Println(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	trimmedText := strings.Trim(text, "\n")
	return trimmedText
}

func (cliController *CLIController) print(msg string) {
	cliController.lock()
	fmt.Println(msg)
	cliController.unlock()
}

func (cliController *CLIController) lock() {
	cliController.userIO.Lock()
}

func (cliController *CLIController) unlock() {
	cliController.userIO.Unlock()
}

func (cliController *CLIController) getInputUnsafe(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt)
	text, _ := reader.ReadString('\n')
	return text[0 : len(text)-1]
}

func (cliController *CLIController) printUnsafe(msg string) {
	fmt.Println(msg)
}

func (cliController *CLIController) startCli(folder FolderManager) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		trimmedText := strings.Trim(text, "\n")
		if cliController.ioWait {
			cliController.inputChan <- trimmedText
			continue
		}
		switch trimmedText {
		case "add":
			fmt.Println("Asking for folder path")
			folderPath := cliController.getCommandInput("Enter the folder path to be added:")
			folder.add(folderPath)
		case "sync":
			folderPath := cliController.getCommandInput("Enter the folder path to be synced")
			folder.sync(folderPath)
			cliController.print("Syncing " + folderPath)
		case "diff":
			folderPath := cliController.getCommandInput("Enter the folder path to be synced")
			folder.sync(folderPath)
			cliController.print("Syncing " + folderPath)
		default:
			if cliController.ioWait {
				fmt.Println("Ignoring ", text)
			} else {
				fmt.Println("Invalid command ", text)
			}
		}
	}
}
