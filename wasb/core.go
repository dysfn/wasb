package wasb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/net/websocket"
)

const (
	slackURLRTMStart = "https://slack.com/api/rtm.start?token=XXX"
	slackURLOrigin   = "https://api.slack.com/"
)

type Cfg struct {
	APIToken string `json:"apitoken"`
	Workers  int    `json:"workers"`
}

type RespRTMStart struct {
	OK    bool              `json:"ok"`
	Error string            `json:"error"`
	URL   string            `json:"url"`
	Self  *RespRTMStartSelf `json:"self"`
}

type RespRTMStartSelf struct {
	ID string `json:"id"`
}

type Msg struct {
	ID      uint64 `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

type WASB interface {
	ReceiveMessage() (*Msg, error)
	IsValidMessage(m *Msg) bool
	SendMessage(m *Msg) error
	TearDown() error
}

func GetCfg(filename string) (*Cfg, error) {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg Cfg
	err = json.Unmarshal(f, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func StartRTM(token string) (*RespRTMStart, error) {
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
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result RespRTMStart
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf(result.Error)
	}

	return &result, nil
}

func GetWSConn(url string) (*websocket.Conn, error) {
	conn, err := websocket.Dial(url, "", slackURLOrigin)
	return conn, err
}

func Start(wasb WASB, workers int) {
	// Wait group to keep track of worker cancellation
	var wg sync.WaitGroup

	// Channel for receiving OS error signals
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close(sigs)

	// Channel for receiving messages
	msgs := make(chan *Msg)

	// Channel for broadcasting "done" signals
	done := make(chan bool)

	// Publish messages
	go func() {
		defer close(msgs)
		for {
			select {
			case <-done:
				return
			default:
				m, err := wasb.ReceiveMessage()
				if err != nil {
					continue
				}
				if wasb.IsValidMessage(m) {
					msgs <- m
				}
			}
		}
	}()

	// Subscribe messages
	startWorker := func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case m := <-msgs:
				err := wasb.SendMessage(m)
				if err != nil {
					continue
				}
			}
		}
	}

	// Start concurrent workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go startWorker()
	}

	// Receive OS error signal
	<-sigs

	// Close channel to broadcast done signals to all worker goroutines
	close(done)

	// Wait for goroutines to complete then terminate
	wg.Wait()

	// Tear down to complete the process
	err := wasb.TearDown()
	if err != nil {
		panic(err)
	}
}
