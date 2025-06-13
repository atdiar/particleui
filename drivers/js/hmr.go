//go:build server

package doc

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"sync"
	"time"
)

// EventCallback is a function type for handling fsnotify events
type EventCallback func(event fsnotify.Event)

// WatchDir starts watching the specified directory and its subdirectories for changes.
func WatchDir(root string, callback func(event fsnotify.Event)) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Walk through the root directory to set up initial watches
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		watcher.Close()
		return nil, err
	}

	// Goroutine for handling all events
	go func() {
		var debounceTimer *time.Timer
		debounceDuration := 100 * time.Millisecond // Set your debounce duration
		var mu sync.Mutex
		var lastEvent fsnotify.Event

		triggerCallback := func() {
			mu.Lock()
			defer mu.Unlock()
			callback(lastEvent)
			debounceTimer = nil
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Automatically add newly created directories to the watcher
				if event.Has(fsnotify.Create) {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						watcher.Add(event.Name)
					}
				}
				log.Println("New fs event:", event) // DEBUG
				mu.Lock()
				lastEvent = event // Update the last event
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, triggerCallback)
				mu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	return watcher, nil
}

type message struct {
	event string
	data  string
	id    string
	retry string
}

func Msg(event, data, id, retry string) message {
	return message{event, data, id, retry}
}

func (m message) String() string {
	eventstr := "event: " + m.event + "\n"
	datastr := "data: " + m.data + "\n"
	idstr := "id: " + m.id + "\n"
	retrystr := "retry: " + m.retry + "\n"
	endstr := "\n\n"
	msgstr := condVal(m.event != "", eventstr, "") + condVal(m.data != "" || (m.event != "" && m.data == ""), datastr, "") + condVal(m.id != "", idstr, "") + condVal(m.retry != "", retrystr, "") + endstr
	return msgstr
}

func condVal[T any](condition bool, valueIfTrue T, valueIfFalse T) T {
	if condition {
		return valueIfTrue
	}
	return valueIfFalse
}

type SSEController struct {
	// The channel to send messages to the client
	Message chan string
	Once    *sync.Once
}

func NewSSEController() *SSEController {
	return &SSEController{
		Message: make(chan string),
		Once:    &sync.Once{},
	}
}

func (s *SSEController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Make sure that the writer supports flushing.
	fw, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	fmt.Println("New client connected, ready to receive messages...")

	// Set the headers related to SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientMsgChan := make(chan string)
	clientClosed := make(chan struct{})

	// Goroutine to handle sending messages to this client
	go func() {
		for {
			select {
			case msg, ok := <-clientMsgChan:
				if !ok {
					return // Exit goroutine if channel is closed
				}
				fmt.Fprintf(w, "%s", msg)
				fw.Flush()
			case <-clientClosed:
				return // Exit goroutine if client is disconnected
			}
		}
	}()

	for {
		select {
		case <-r.Context().Done():
			fmt.Println("Connection closed from client")
			close(clientClosed)  // Notify the sending goroutine to stop
			close(clientMsgChan) // Close the channel to stop sending messages
			return
		case msg := <-s.Message:
			// Send the message to the client-specific channel
			clientMsgChan <- msg
		}
	}
}

func (s *SSEController) SendEvent(event, data, id, retry string) {
	s.Message <- Msg(event, data, id, retry).String()
}
