package main

import (
	"log"

	"github.com/microamp/wasb/wasb"

	"golang.org/x/net/websocket"
)

const configFile = "config.json"

// Echo ...
type Echo struct {
	conn *websocket.Conn
}

// ReceiveMessage ...
func (bot *Echo) ReceiveMessage() (*wasb.Msg, error) {
	var m wasb.Msg
	err := websocket.JSON.Receive(bot.conn, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// FilterMessage ...
func (bot *Echo) FilterMessage(m *wasb.Msg) bool {
	return m.Type == "message" && m.Text != ""
}

// SendMessage ...
func (bot *Echo) SendMessage(m *wasb.Msg) error {
	err := websocket.JSON.Send(bot.conn, m)
	return err
}

func main() {
	log.Printf("Loading config...")
	cfg, err := wasb.GetCfg(configFile)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Config loaded")

	log.Printf("Starting RTM...")
	respRTMStart, err := wasb.StartRTM(cfg.APIToken)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("RTM started")

	log.Printf("Establishing Websocket connection...")
	conn, err := wasb.GetWSConn(respRTMStart.URL)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Websocket connection established")
	defer func() {
		err = conn.Close()
		log.Printf("Error closing Websocket connection: %+v", err)
	}()

	log.Printf("Launching the bot...")
	echoBot := &Echo{conn: conn}
	wasb.Start(echoBot, cfg.Workers)
}
