package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func check(t *testing.T, result, expect interface{}) {
	if result != expect {
		t.Errorf("result: %v, expect: %v", result, expect)
	}
}

func TestEcho(t *testing.T) {
	expect := "hello"
	b, err := exec.Command("echo", "hello").Output()
	if err != nil {
		t.Fatal(err)
	}
	result := strings.TrimSpace(string(b))
	check(t, result, expect)
}

func TestLookPath(t *testing.T) {
	executable := "echo"
	_, err := exec.LookPath(executable)

	if err != nil {
		t.Errorf("%s not found in path", executable)
	}
}

func TestNetcat(t *testing.T) {
	expect := "hello"
	executable := "nc"
	_, err := exec.LookPath(executable)

	if err != nil {
		t.Fatalf("%s not found in path", executable)
	}

	port := "8000"
	cmd := exec.Command("nc", "-lk", "127.0.0.1", port)

	stdout, _ := cmd.StdoutPipe()

	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	// give process time to start
	<-time.After(time.Millisecond * 200)

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	done := make(chan bool)

	// read nc stdout
	var result []byte
	var reader = func() {
		result, err = ioutil.ReadAll(stdout)
		if err != nil {
			t.Fatalf("failed to read nc output. %v", err)
		}
		done <- true
	}
	go reader()

	// send string to nc
	w := bufio.NewWriter(conn)
	w.WriteString(expect)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}

	// give time for string to be recieved by nc
	<-time.After(time.Millisecond * 200)

	// kill nc. this causes reader goroutine to continue / finish reading from
	// stdout
	cmd.Process.Kill()

	select {
	case <-done:
		check(t, string(result), expect)
	case <-time.After(time.Second):
		t.Errorf("timeout hit")
	}

}

func TestOpensnoop(t *testing.T) {
	cmdName := "opensnoop"
	//cmdArgs := []string{}

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo %s", cmdName))

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating StdoutPipe for Cmd: %v\nerror: %v\n", cmdName, err)
		os.Exit(1)
	}

	lines := make(chan string)

	// read stdout
	scanner := bufio.NewScanner(cmdReader)
	go func(lines chan string) {
		for scanner.Scan() {
			line := scanner.Text()
			lines <- line
		}
	}(lines)

	// process lines coming in from stdout
	go func(lines chan string) {
		for {
			select {
			case line := <-lines:
				fmt.Printf("%s\n", line)
			default:
			}
		}
	}(lines)

	// start process
	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}

	// wait for process to exit
	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		os.Exit(1)
	}
}
