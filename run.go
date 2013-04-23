package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/drone/routes"
	"github.com/garyburd/redigo/redis"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var addr = flag.String("addr", ":8088", "http service address")

var homeTempl = template.Must(template.ParseFiles("templates/task.html"))

var redisServer = "localhost:6379"
var pool = &redis.Pool{
	MaxIdle:     3,
	IdleTimeout: 240 * time.Second,
	Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", redisServer)
		if err != nil {
			return nil, err
		}
		return c, err
	},
	TestOnBorrow: func(c redis.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	},
}

type TaskExecution struct {
	Service string `json:"service"`
	Task    string `json:"task"`
	TaskId  string `json:"task_id"`
}

func (task *TaskExecution) RedisKey() string {
	return fmt.Sprintf("task:%s", task.TaskId)
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

func executeRoute(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	task := &TaskExecution{params.Get(":service"), params.Get(":task"), generateId()}
	go executeTask(task)
	http.Redirect(w, r, fmt.Sprintf("/task/%s", task.TaskId), 302)
}

func taskRoute(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	conn := pool.Get()
	defer conn.Close()
	var buffer bytes.Buffer

	// TODO get list and read all lines
	// TODO allow client to specify offset
	// TODO send client next key to look for
	task := &TaskExecution{TaskId: params.Get(":taskId")}
	lines, err := redis.Strings(conn.Do("LRANGE", task.RedisKey(), 0, 4000))
	var cmdOut string
	if err != nil {
		cmdOut = err.Error()
	} else {
		for _, line := range lines {
			buffer.WriteString(line)
		}
		cmdOut = buffer.String()
	}
	ctx := map[string]string{
		"cmdOut": cmdOut,
	}
	homeTempl.Execute(w, ctx)
}

func startWebserver() {
	mux := routes.New()
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	mux.Static("/static/", cwd)
	mux.Get("/execute/:service/:task", executeRoute)
	mux.Get("/task/:taskId", taskRoute)

	http.Handle("/", mux)
	log.Printf("http server started on port %s", *addr)
	http.ListenAndServe(*addr, nil)
}

func executeTask(task *TaskExecution) {
	cmd := exec.Command("/bin/bash", "test.sh")
	commandOut := make(chan string)
	defer close(commandOut)
	go printOutput(task, commandOut)
	captureOutput(cmd, commandOut)
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
	// TODO write status code to output
	return cmd.Wait()
}

func printOutput(task *TaskExecution, commandOut chan string) {
	conn := pool.Get()
	defer conn.Close()
	fmt.Println(task.RedisKey())
	for line := range commandOut {
		fmt.Println(line)
		conn.Do("RPUSH", task.RedisKey(), line)
	}
}

func main() {
	flag.Parse()
	startWebserver()
}
