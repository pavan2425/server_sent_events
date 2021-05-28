package main

import (
	"bytes"
	"encoding/json"
	"text/template"

	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/v3/mem"
)

type cpuLoad struct {
	Field        string
	MemoryStatus memoryDetails
}
type memoryDetails struct {
	Total       uint64
	Free        uint64
	Usedpercent float64
}

func handleMemorySse(w http.ResponseWriter, r *http.Request) {
	log.Printf("Get handshake from client")

	// prepare the header
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// instantiate the channel
	messageChan := make(chan cpuLoad)
	go sendMessageToChannel(messageChan)
	// close the channel after exit the function
	defer func() {
		close(messageChan)
		messageChan = nil
		log.Printf("client connection is closed")
	}()

	// prepare the flusher
	flusher, _ := w.(http.Flusher)
	// trap the request under loop forever
	for {

		select {

		// message will received here and printed
		case message := <-messageChan:
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.Encode(message)
			fmt.Fprintf(w, "data: %v\n\n", buf.String())

			flusher.Flush()

		// connection is closed then defer will be executed
		case <-r.Context().Done():
			return

		}
	}
}

func startPage(w http.ResponseWriter, r *http.Request) {

	parsedTemplate, _ := template.ParseFiles("index.html")
	err := parsedTemplate.Execute(w, "Memory chart")
	if err != nil {
		log.Println("Error executing template :", err)
		return
	}
}
func sendMessageToChannel(messageChan chan cpuLoad) {
	for {
		if messageChan != nil {
			memory, err := mem.VirtualMemory()

			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			details := memoryDetails{
				Total:       memory.Total,
				Free:        memory.Free,
				Usedpercent: memory.UsedPercent,
			}

			cpudetails := cpuLoad{
				Field:        "memory",
				MemoryStatus: details,
			}
			messageChan <- cpudetails
			// send the message through the available channel
		}
		time.Sleep(4 * time.Second)
	}

}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", startPage)
	router.HandleFunc("/getmemorydetails", handleMemorySse)
	// router.HandleFunc("/sendmessage", SendMessage)

	log.Fatal(http.ListenAndServe(":9000", router))
}
