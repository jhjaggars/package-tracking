package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	fmt.Println("Testing Server Signal Behavior...")

	tests := []struct {
		name     string
		signal   os.Signal
		expected string
	}{
		{"SIGTERM (graceful)", syscall.SIGTERM, "Should shutdown gracefully"},
		{"SIGINT (Ctrl+C)", syscall.SIGINT, "Should shutdown gracefully"},
		{"SIGKILL (force)", syscall.SIGKILL, "Should terminate immediately (uncatchable)"},
	}

	for _, test := range tests {
		fmt.Printf("\n=== Testing %s ===\n", test.name)
		testSignal(test.signal, test.expected)
	}
}

func testSignal(sig os.Signal, expected string) {
	fmt.Printf("Expected behavior: %s\n", expected)

	// Start server
	cmd := exec.Command("go", "run", "cmd/server/main.go")
	cmd.Env = append(os.Environ(), "SERVER_PORT=8081") // Use different port
	
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start server: %v", err)
		return
	}

	pid := cmd.Process.Pid
	fmt.Printf("Started server with PID: %d\n", pid)

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Send signal
	fmt.Printf("Sending %s to PID %d...\n", sig, pid)
	start := time.Now()
	
	if err := cmd.Process.Signal(sig); err != nil {
		log.Printf("Failed to send signal: %v", err)
		cmd.Process.Kill()
		return
	}

	// Wait for process to finish
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		duration := time.Since(start)
		fmt.Printf("Process ended after %v\n", duration)
		if err != nil {
			fmt.Printf("Process exit error: %v\n", err)
		} else {
			fmt.Printf("Process exited cleanly\n")
		}
	case <-time.After(10 * time.Second):
		fmt.Printf("Process didn't exit within 10 seconds\n")
		cmd.Process.Kill()
	}
}