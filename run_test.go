package main

import (
	"fmt"
	"github.com/bmizerany/assert"
	"os/exec"
	"testing"
)

func outputToSlice(lines *[]string, cmdOut chan string) {
	for s := range cmdOut {
		fmt.Println(s)
		*lines = append(*lines, s)
	}
}

func testCommand(t *testing.T, cmd *exec.Cmd, expected []string) {
	commandOut := make(chan string)
	lines := make([]string, 0)
	go outputToSlice(&lines, commandOut)
	err := captureOutput(cmd, commandOut)
	assert.Equal(t, nil, err)
	assert.Equal(t, lines, expected)
}

func TestNoNewline(t *testing.T) {
	cmd := exec.Command("echo", "-n", "hello")
	expected := []string{"hello"}
	testCommand(t, cmd, expected)
}

func testMultiline(t *testing.T) {
	cmd := exec.Command("echo", "-e", "hello\nworld")
	expected := []string{"hello\nworld\n"}
	testCommand(t, cmd, expected)
}
