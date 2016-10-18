package main

import (
	"log"

	"golang.org/x/net/websocket"
)

// Echo ...
type Echo struct {
	conn *websocket.Conn
}

// ReceiveMessage ...
func (bot *Echo) ReceiveMessage() (*Msg, error) {
	var m Msg
	err := websocket.JSON.Receive(bot.conn, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// FilterMessage ...
func (bot *Echo) FilterMessage(m *Msg) bool {
	return m.Type == "message" && m.Text != ""
}

// SendMessage ...
func (bot *Echo) SendMessage(m *Msg) error {
	err := websocket.JSON.Send(bot.conn, m)
	return err
}

func main() {
	log.Printf("Loading config...")
	cfg, err := GetCfg(configFile)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Config loaded")

	log.Printf("Starting RTM...")
	respRTMStart, err := StartRTM(cfg.APIToken)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("RTM started")

	log.Printf("Establishing Websocket connection...")
	conn, err := GetWSConn(respRTMStart.URL)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Websocket connection established")
	defer func() {
		err = conn.Close()
		log.Printf("Error closing Websocket connection: %+v", err)
	}()

	echoBot := &Echo{conn: conn}
	Start(echoBot, cfg.Workers)
}
