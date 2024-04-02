/*
@author Thomas Hutton
This is a simple shell that takes user input and can perform cd, ls, whoami,
wc, mkdir, cp, and mv commands from the OS. Can also type exit to exit the
shell. cd and whoami are run natively from this program while the rest are
run throuh the exec.Command function from os/exec
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"project1/filesystem"
	"strings"
)

func main() {
	filesystem.InitializeDisk()
	disk := filesystem.VirtualDisk
	inodes := filesystem.ReadInodesFromDisk()
	fmt.Println(filesystem.LastInodeBlock)
	fmt.Println(inodes)
	filesystem.Open("open", "hello.jpg", 1)
	filesystem.Open("open", "hello.jpg", 1)
	filesystem.Open("open", "hey.txt", 1)
	filesystem.Open("open", "hey.txt", 1)
	filesystem.Open("open", "yog.txt", 1)
	filesystem.Open("open", "yoh.txt", 1)
	filesystem.Open("open", "yod.txt", 1)
	filesystem.Open("open", "yos.txt", 1)
	filesystem.Open("open", "yoa.txt", 1)
	filesystem.Open("open", "yot.txt", 1)
	filesystem.Open("open", "yop.txt", 1)
	filesystem.Open("open", "yoo.txt", 1)
	filesystem.Open("open", "yoi.txt", 1)
	filesystem.Open("open", "you.txt", 1)
	filesystem.Open("open", "yom.txt", 1)
	filesystem.Open("open", "yon.txt", 1)
	filesystem.Open("open", "yob.txt", 1)
	disk = filesystem.VirtualDisk
	filesystem.Open("open", "yov.txt", 1)
	filesystem.Open("open", "yoc.txt", 1)
	filesystem.Open("open", "yos.txt", 1)
	inodes = filesystem.ReadInodesFromDisk()
	disk = filesystem.VirtualDisk
	fmt.Println(inodes)
	fmt.Println(disk)
	fmt.Println("what the heck")
	//got info about bufio and strings from here https://tutorialedge.net/golang/reading-console-input-golang/
	//create scanner
	scanner := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to shell, please enter commands")
	//for loop, each iteration simulates one line of shell
	for {
		//create userinput using the scanner.ReadString, delimiter newline
		userinput, _ := scanner.ReadString('\n')
		list := strings.Fields(userinput)
		switch list[0] {
		//first case exit, exits the shell
		case "exit":
			os.Exit(0)
		//case cd,
		case "cd":
			//only cd was typed
			if len(list) < 2 {
				//Got this from here https://stackoverflow.com/questions/46028707/how-to-change-the-current-directory-in-go
				home, _ := os.UserHomeDir()
				err := os.Chdir(home)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				//cd plus a directory was typed
				os.Chdir(list[1])
			}
		//case whoami prints name
		case "whoami":
			fmt.Println("thutton2 Thomas Hutton")
		//case ls uses exec.Command to issue the ls command
		case "ls":
			// This command I understood from here https://stackoverflow.com/questions/22781788/how-could-i-pass-a-dynamic-set-of-arguments-to-gos-command-exec-command
			command := exec.Command("ls", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		//case wc uses exec.Command to issue wc command
		case "wc":
			command := exec.Command("wc", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		//case mkdir uses exec.Command to issue mkdir command
		case "mkdir":
			command := exec.Command("mkdir", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		//case cp uses exec.Command to issue cp command
		case "cp":
			command := exec.Command("cp", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		//case mv uses exec.Command to issue mv command
		case "mv":
			command := exec.Command("mv", list[1:]...)
			out, err := command.Output()
			if err != nil {
				fmt.Println("Need valid args")
			} else {
				fmt.Println(string(out))
			}
		//default returns "invalid command" string
		default:
			fmt.Println("Invalid Command")
		}

	}

}
