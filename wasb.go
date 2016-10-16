package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/websocket"
)

const (
	configFile       = "config.json"
	slackURLRTMStart = "https://slack.com/api/rtm.start?token=XXX"
	slackURLOrigin   = "https://api.slack.com/"
)

// Config ...
type Config struct {
	APIToken string `json:"apitoken"`
	Workers  int    `json:"workers"`
}

// RespRTMStart ...
type RespRTMStart struct {
	OK    bool              `json:"ok"`
	Error string            `json:"error"`
	URL   string            `json:"url"`
	Self  *RespRTMStartSelf `json:"self"`
}

// RespRTMStartSelf ...
type RespRTMStartSelf struct {
	ID string `json:"id"`
}

// Message ...
type Message struct {
	ID      uint64 `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func startRTM(token string) (*RespRTMStart, error) {
	u, err := url.Parse(slackURLRTMStart)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	req := &http.Request{
		Method: "GET",
		URL:    u,
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = resp.Body.Close()
		log.Printf("Error closing response body: %+v", err)
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result RespRTMStart
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalln(err)
	}
	if !result.OK {
		return nil, fmt.Errorf(result.Error)
	}

	return &result, nil
}

func startWorker(wg *sync.WaitGroup, done <-chan bool, msgs <-chan Message) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case <-done:
			return
		case m := <-msgs:
			log.Printf("Received: %s", m.Text)
			time.Sleep(10 * time.Second)
		}
	}
}

func main() {
	f, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
	}
	var config Config
	err = json.Unmarshal(f, &config)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Starting RTM...")
	respRTMStart, err := startRTM(config.APIToken)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("RTM started")

	log.Printf("Establishing Websocket connection...")
	wsConn, err := websocket.Dial(respRTMStart.URL, "", slackURLOrigin)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Websocket connection established")
	defer func() {
		err = wsConn.Close()
		log.Printf("Error closing Websocket connection: %+v", err)
	}()

	// Channel for receiving OS error signals
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close(sigs)

	// Channel for receiving messages
	msgs := make(chan Message)
	defer close(msgs)

	// Channel for broadcasting "done" signals
	done := make(chan bool, config.Workers)
	defer close(done)

	log.Printf("Receiving messages...")
	go func() {
		for {
			var m Message
			err = websocket.JSON.Receive(wsConn, &m)
			if err != nil {
				log.Printf("Error receiving message: %+v", err)
			}
			if m.Type == "message" {
				msgs <- m
			}
		}
	}()

	var wg sync.WaitGroup

	// Start concurrent workers
	for i := 0; i < config.Workers; i++ {
		go startWorker(&wg, done, msgs)
	}

	// Wait for workers to complete then terminate
	for s := range sigs {
		log.Printf("Signal (%s) received. Waiting for workers to complete...", s)

		// Broadcost done signals to all worker goroutines
		for i := 0; i < config.Workers; i++ {
			done <- true
		}

		wg.Wait()
		log.Fatalf("Aborting...")
	}
}
