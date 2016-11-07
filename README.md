# wasb

[![Build Status](https://travis-ci.org/dysfn/wasb.svg?branch=master)](https://travis-ci.org/dysfn/wasb)

Write a Slack bot in Go

## Quickstart

1. Create a new bot user on Slack.
2. Prepare a new config file for the bot.

        cp config.json.template echo-config.json
3. Update the config with the bot's API token.
4. Run the bot by running the following command.

        go run cmd/echo/echo.go -config=echo-config.json

## Write your own

Your custom bot needs to implement the `WASB` interface as shown below.

```go
type WASB interface {
	ReceiveMessage() (*Msg, error)
	IsValidMessage(m *Msg) bool
	SendMessage(m *Msg) error
	TearDown() error
}
```

See how `Echo` implements it in [`cmd/echo/echo.go`](https://github.com/dysfn/wasb/blob/master/cmd/echo/echo.go).

## License

MIT
