package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/drone/routes"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var addr = flag.String("addr", ":8088", "http service address")

type TaskExecution struct {
	Service string `json:"service"`
	Task    string `json:"task"`
	TaskId  string `json:"task_id"`
}

type logLine struct {
	task TaskExecution
	line string
}

func generateId() string {
	buf := make([]byte, 16)
	io.ReadFull(rand.Reader, buf)
	return fmt.Sprintf("%x", buf)
}

func taskRoute(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	task := &TaskExecution{params.Get(":service"), params.Get(":task"), generateId()}
	executeTask(task)
	routes.ServeJson(w, task)
}

func startWebserver() {
	mux := routes.New()
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	mux.Static("/static/", cwd)
	mux.Get("/api/:service/:task", taskRoute)

	http.Handle("/", mux)
	http.ListenAndServe(*addr, nil)
}

func executeTask(task *TaskExecution) {
	cmd := exec.Command("echo", "-n", "run.go\ntest\ntest\n")
	commandOut := make(chan string)
	go printOutput(commandOut)
	captureOutput(cmd, commandOut)
	close(commandOut)
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
	for line := range commandOut {
		fmt.Printf("%s", line)
	}
}

func main() {
	flag.Parse()
	startWebserver()
}
