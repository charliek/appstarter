package main

import (
	"bufio"
	"fmt"
	"github.com/drone/routes"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func Whoami(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	lastName := params.Get(":last")
	firstName := params.Get(":first")
	fmt.Fprintf(w, "you are %s %s", firstName, lastName)
}

func startWebserver() {
	mux := routes.New()
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cwd)
	mux.Static("/static/", cwd)
	mux.Get("/:last/:first", Whoami)

	http.Handle("/", mux)
	http.ListenAndServe(":8088", nil)
}

func captureOutput(cmd *exec.Cmd, commandOut chan string) error {
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout := bufio.NewReader(outPipe)
	cmd.Start()

	for {
		line, err := stdout.ReadString('\n')
		if err == nil || err == io.EOF {
			if len(line) > 0 {
				commandOut <- line
			}
		}
		if err != nil {
			break
		}
	}
	return cmd.Wait()
}

func printOutput(commandOut chan string) {
	for s := range commandOut {
		fmt.Print(s)
	}
}

func main() {
	commandOut := make(chan string)
	cmd := exec.Command("echo", "-n", "run.go")
	go printOutput(commandOut)
	captureOutput(cmd, commandOut)
	close(commandOut)
	startWebserver()
}
