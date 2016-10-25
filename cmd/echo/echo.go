package main

import (
	"flag"
	"log"
	"os"

	"github.com/microamp/wasb/wasb"

	"golang.org/x/net/websocket"
)

const defaultConfigFile = "config.json"

var configFile string

type Echo struct {
	conn *websocket.Conn
}

func (bot *Echo) ReceiveMessage() (*wasb.Msg, error) {
	var m wasb.Msg
	err := websocket.JSON.Receive(bot.conn, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (bot *Echo) IsValidMessage(m *wasb.Msg) bool {
	return m.Type == "message" && m.Text != ""
}

func (bot *Echo) SendMessage(m *wasb.Msg) error {
	err := websocket.JSON.Send(bot.conn, m)
	return err
}

func (bot *Echo) TearDown() error {
	log.Printf("Closing websocket connection...")
	err := bot.conn.Close()
	return err
}

func main() {
	flags := flag.NewFlagSet("echo", flag.ExitOnError)
	flags.StringVar(&configFile, "config", defaultConfigFile, "")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Loading config (filename: %s)...", configFile)
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

	log.Printf("Launching the bot...")
	echoBot := &Echo{conn: conn}
	wasb.Start(echoBot, cfg.Workers)
}
