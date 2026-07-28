package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func main() {
	dir, err := os.MkdirTemp("", "fsnotify-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0755); err != nil {
		panic(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		panic(err)
	}

	done := make(chan bool)
	go func() {
		select {
		case ev := <-watcher.Events:
			fmt.Printf("Event received: %v\n", ev)
			done <- true
		case err := <-watcher.Errors:
			fmt.Printf("Error received: %v\n", err)
			done <- false
		case <-time.After(2 * time.Second):
			fmt.Println("Timeout! No event received.")
			done <- false
		}
	}()

	time.Sleep(500 * time.Millisecond)

	testFile := filepath.Join(subdir, "file.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	success := <-done
	if success {
		fmt.Println("Recursive: YES")
	} else {
		fmt.Println("Recursive: NO")
	}
}
