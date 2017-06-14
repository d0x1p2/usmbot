package main

import (
	"fmt"
	"os"

	"github.com/d0x1p2/godbot"
)

// Constants used to initiate and customize bot.
var (
	_version = "0.1.2"
	envToken = os.Getenv("BOT_TOKEN")
	envDBUrl = os.Getenv("BOT_DBURL")
)

func main() {

	var binfo bot

	if envToken == "" {
		return
	}

	bot, err := godbot.New(envToken)
	if err != nil {
		fmt.Println(err)
		return
	}

	bot.MessageHandler(msghandler)
	err = bot.Start()
	if err != nil {
		fmt.Println(err)
	}

	binfo.Core = bot
	binfo.core()

}

func (b *bot) cleanup() {
	b.Stop()
	fmt.Println("Bot stopped, exiting.")
	os.Exit(0)
}
