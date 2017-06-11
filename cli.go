package main

import (
	"bufio"
	"fmt"
	"sync"
	"os"
)

type CLIController struct{
	userIO sync.Mutex
}

func (cliController CLIController) getInput(prompt string) string{
	cliController.lock()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt)
	text, _ := reader.ReadString('\n')
	cliController.unlock()
	return text
}

func (cliController CLIController) lock(){
	cliController.userIO.Lock()
}

func (cliController CLIController) unlock(){
	cliController.userIO.Unlock()
}