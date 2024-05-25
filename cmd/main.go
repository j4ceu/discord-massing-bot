package main

import "MassingDiscordBot/internal/bot"

func main() {
	bot.BotToken = ""
	bot.ApplicationID = ""
	bot.Run() // call the run function of bot/bot.go
}
