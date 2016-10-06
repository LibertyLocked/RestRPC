package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/satori/go.uuid"
)

// WebInput represents an input request sent by web clients
type WebInput struct {
	Header byte   // The header of the request
	Target string `json:",omitempty"` // The target WSH component this request is going to. This field is not sent to the component
	Cmd    string
	Args   []interface{} `json:",omitempty"`
	UID    uuid.UUID     // UID attached to this request. Used to identify the response to this request when the response is sent over socket
}

type componentRecord struct {
	Name   string
	Remote string
}

var inChMap = make(map[string]chan WebInput)        // Input channel map, used to send inputs from web to components. Key is component ID
var retChMap = make(map[uuid.UUID]chan interface{}) // Return data map, used for WSH component return values

func handleStatusGet(w http.ResponseWriter, r *http.Request) {
	// Lists all the components connected and their remote endpoints
	componentList := []componentRecord{}
	for k, v := range componentNameMap {
		componentList = append(componentList, componentRecord{v, k.RemoteAddr})
	}
	jsonOutput, err := json.Marshal(componentList)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
	}
	w.Write(jsonOutput)
}

func handleInputPost(w http.ResponseWriter, r *http.Request) {
	// Front-end client sends input to game
	decoder := json.NewDecoder(r.Body)
	var input WebInput
	err := decoder.Decode(&input)
	if err != nil {
		log.Println("POST: Error decoding input:", err)
		return
	}
	// Attach a UUID to the request input
	input.UID = uuid.NewV4()
	// Get the target of the request, then clear it from the input so we don't send it to component
	targetID := input.Target
	input.Target = ""

	select {
	case inChMap[targetID] <- input:
		log.Println("POST: Sent:", input)
		// Make a channel and wait for the return value for this input
		retChMap[input.UID] = make(chan interface{}, 1)
		defer func() {
			delete(retChMap, input.UID)
		}()
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(1 * time.Second)
			timeout <- true
		}()
		// Now we wait till the component sends the return value back, or it times out
		select {
		case retVal := <-retChMap[input.UID]:
			// A return value has arrived!
			seralizedRet, err := json.Marshal(retVal)
			if err != nil {
				log.Println("POST: Return marshal error:", err)
			} else {
				// Return value successfully retrieved
				log.Println("POST: Returned:", string(seralizedRet))
				w.Write(seralizedRet)
			}
		case <-timeout:
			// Return value did not arrive in time
			log.Println("POST: Return timeout:", input.UID)
			http.Error(w, http.StatusText(504), 504)
		}
	default:
		// Fails to POST the input because component's chan is full, or its channel does not exist
		log.Println("POST: Input channel unavailable. Discarding:", input)
		http.Error(w, http.StatusText(502), 502)
	}
}

func handleFetchGet(w http.ResponseWriter, r *http.Request) {

}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies
	files, _ := ioutil.ReadDir("./apps/")
	tmplData := map[string]time.Time{}
	for _, f := range files {
		if f.IsDir() {
			tmplData[f.Name()] = f.ModTime()
		}
	}
	renderTemplate(w, "index.html", tmplData)
}