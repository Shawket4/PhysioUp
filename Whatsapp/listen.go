package Whatsapp

import (
	"fmt"

	whatsapp_chatbot_golang "github.com/green-api/whatsapp-chatbot-golang"
)

func Listen() {
	bot := whatsapp_chatbot_golang.NewBot("7105184364", "2a07907c964d418f99322d9769ae79439269f69b183741a7b6")

	bot.SetStartScene(StartScene{})

	bot.StartReceivingNotifications()
}

type StartScene struct {
}

func (s StartScene) Start(bot *whatsapp_chatbot_golang.Bot) {
	bot.IncomingMessageHandler(func(message *whatsapp_chatbot_golang.Notification) {
		text, _ := message.Text()
		fmt.Println(text)
		// message.AnswerWithText(text)
	})
}
