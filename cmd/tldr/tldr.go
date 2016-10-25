package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/microamp/go-smmry/smmry"
	"github.com/microamp/wasb/wasb"

	"golang.org/x/net/websocket"
)

const defaultConfigFile = "config.json"

var configFile string

type TLDR struct {
	conn          *websocket.Conn
	summaryLength string
	botID         string
}

func (bot *TLDR) ReceiveMessage() (*wasb.Msg, error) {
	var m wasb.Msg
	err := websocket.JSON.Receive(bot.conn, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (bot *TLDR) IsValidMessage(m *wasb.Msg) bool {
	return m.Type == "message" &&
		strings.HasPrefix(m.Text, fmt.Sprintf("<@%s> <", bot.botID)) &&
		strings.HasSuffix(m.Text, ">")
}

func (bot *TLDR) SendMessage(m *wasb.Msg) error {
	client, err := smmry.NewSmmryClient()
	if err != nil {
		return err
	}

	url := strings.TrimLeft(m.Text, fmt.Sprintf("<@%s> ", bot.botID))
	url = strings.TrimRight(url, ">")
	summary, err := client.SummaryByWebsite(url, bot.summaryLength)
	if err != nil {
		return err
	}

	resp := &wasb.Msg{
		ID:      m.ID,
		Type:    "message",
		Channel: m.Channel,
		Text:    summary.SmAPIContent,
	}
	err = websocket.JSON.Send(bot.conn, resp)
	return err
}

func (bot *TLDR) TearDown() error {
	log.Printf("Closing websocket connection...")
	err := bot.conn.Close()
	return err
}

func main() {
	flags := flag.NewFlagSet("tl;dr", flag.ExitOnError)
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

	log.Printf("Launching the bot...")
	tldrBot := &TLDR{
		conn:          conn,
		summaryLength: "5",
		botID:         respRTMStart.Self.ID,
	}
	wasb.Start(tldrBot, cfg.Workers)
}
