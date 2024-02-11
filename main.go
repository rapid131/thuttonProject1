package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	scanner := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(">>>")
		userinput, _ := scanner.ReadString('\n')
		args := strings.Fields(userinput)
		switch args[0] {
		case "exit":
			os.Exit(0)
		case "cd":
			if len(args) < 2 {
				//Got this from here https://stackoverflow.com/questions/46028707/how-to-change-the-current-directory-in-go
				home, _ := os.UserHomeDir()
				err := os.Chdir(home)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				os.Chdir(args[1])
			}
		case "whoami":
			fmt.Println("thutton2 Thomas Hutton")
		default:
			// This command I understood from here https://stackoverflow.com/questions/22781788/how-could-i-pass-a-dynamic-set-of-arguments-to-gos-command-exec-command
			command := exec.Command(args[0], args[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args or command invalid")
			}
			fmt.Println(string(out))
		}

	}

}
