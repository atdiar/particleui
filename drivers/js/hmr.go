//go:build server

package doc

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "net/http"
    "github.com/fsnotify/fsnotify"

    "sync"
)

var SSEChannel = NewSSEController()

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
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                // Automatically add newly created directories to the watcher
                if event.Op&fsnotify.Create == fsnotify.Create {
                    info, err := os.Stat(event.Name)
                    if err == nil && info.IsDir() {
                        watcher.Add(event.Name)
                    }
                }
                callback(event) // Invoke the callback for every event

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
    msgstr := valif(m.event != "", eventstr, "") + valif(m.data != "" || (m.event != "" && m.data == ""), datastr, "") + valif(m.id != "", idstr, "") + valif(m.retry != "", retrystr, "") + endstr
    return msgstr
}


func valif[T any](condition bool, valueIfTrue T, valueIfFalse T) T {
    if condition {
        return valueIfTrue
    }
    return valueIfFalse
}

type SSEController struct{
    // The channel to send messages to the client
    Message chan string
    Once sync.Once
}

func NewSSEController() *SSEController {
    return &SSEController{
        Message: make(chan string),
    }
}

func(s *SSEController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Make sure that the writer supports flushing.
    fw, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

    // Set the headers related to SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // Listen to the relevant channels for events
    for {
        select {
        case <-r.Context().Done():
            // Close the channel if the connection is closed
            s.Once.Do(func(){
                fmt.Fprintf(w, "SSE Channel closed")
                fmt.Print("Connection closed. SSE channel will close as well")
                close(s.Message)
                fw.Flush()
            })
            return
        case msg := <-s.Message:
            // Send the message to the client
            fmt.Fprintf(w, "%s", msg)
            fw.Flush()
        }
    }
}

func(s *SSEController) SendEvent(event, data, id, retry string) {
    s.Message <- Msg(event, data, id, retry).String()
}