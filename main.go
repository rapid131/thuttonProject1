package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	//create scanner
	scanner := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to shell, please enter commands")
	//for loop
	for {
		userinput, _ := scanner.ReadString('\n')
		list := strings.Fields(userinput)
		switch list[0] {
		case "exit":
			os.Exit(0)
		case "cd":
			if len(list) < 2 {
				//Got this from here https://stackoverflow.com/questions/46028707/how-to-change-the-current-directory-in-go
				home, er := os.UserHomeDir()
				err := os.Chdir(home)
				if er != nil {
					fmt.Println(err)
				}
			} else {
				os.Chdir(list[1])
			}
		case "whoami":
			fmt.Println("thutton2 Thomas Hutton")
		case "ls":
			// This command I understood from here https://stackoverflow.com/questions/22781788/how-could-i-pass-a-dynamic-set-of-arguments-to-gos-command-exec-command
			command := exec.Command("ls", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		case "wc":
			command := exec.Command("wc", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		case "mkdir":
			command := exec.Command("mkdir", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		case "cp":
			command := exec.Command("cp", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		case "mv":
			command := exec.Command("mv", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		default:
			fmt.Println("Invalid Command")
		}

	}

}
