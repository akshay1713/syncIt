package main

import (
	"bufio"
	"fmt"
	"github.com/abiosoft/ishell"
	"os"
	"sync"
)

type CLIController struct {
	userIO sync.Mutex
}

func (cliController *CLIController) getInput(prompt string) string {
	cliController.lock()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt)
	text, _ := reader.ReadString('\n')
	cliController.unlock()
	return text[0 : len(text)-1]
}


func (cliController *CLIController) print(msg string) {
	cliController.lock()
	fmt.Println(msg)
	cliController.unlock()
}

func (cliController *CLIController) lock() {
	cliController.userIO.Lock()
}

func (cliController CLIController) unlock() {
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

func startCli(cliController CLIController, folder FolderManager) {
	shell := ishell.New()
	shell.Println("Started syncIt")
	shell.AddCmd(&ishell.Cmd{
		Name: "add",
		Help: "Add folder to be synced",
		Func: func(c *ishell.Context) {
			folderPath := cliController.getInput("Enter the folder path to be added:")
			folder.add(folderPath)
		},
	})
	shell.AddCmd(&ishell.Cmd{
		Name: "sync",
		Help: "Sync a folder added earlier",
		Func: func(c *ishell.Context) {
			folderPath := cliController.getInput("Enter the folder path to be synced")
			folder.sync(folderPath)
			cliController.print("Syncing " + folderPath)
		},
	})
	shell.Start()
}
