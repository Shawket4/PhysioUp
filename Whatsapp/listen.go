package Whatsapp

import (
	"PhysioUp/Constants"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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

func CheckLogin(c *gin.Context) {
	client := &http.Client{}
	method := "GET"

	url := Constants.WhatsappGoService + "/app/devices"
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	var output struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Results []struct {
			Name   string `json:"name"`
			Device string `json:"device"`
		}
	}
	if err = json.Unmarshal(body, &output); err != nil {
		log.Println(err)
		return
	}

	if len(output.Results) == 0 {
		fmt.Println(output)
		c.JSON(http.StatusOK, gin.H{"message": "Not Logged In"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged In"})
}

func GetQRCode(c *gin.Context) {
	client := &http.Client{}
	method := "GET"

	urlLogin := Constants.WhatsappGoService + "/app/login"
	req, err := http.NewRequest(method, urlLogin, nil)

	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	var output struct {
		Results struct {
			QRLink string `json:"qr_link"`
		} `json:"results"`
	}

	if err = json.Unmarshal(body, &output); err != nil {
		log.Println(err)
	}

	req, err = http.NewRequest(method, output.Results.QRLink, nil)

	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Content-Type", "application/json")

	res, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	c.Header("Content-Disposition", "attachment; filename=qr.png")
	c.Data(http.StatusOK, "application/octet-stream", body)
}

func SendMessage(phone, message string) error {
	client := &http.Client{}
	method := "POST"

	urlLogin := Constants.WhatsappGoService + "/send/message"
	dataStr := fmt.Sprintf(`{"phone": "%s", "message": "%s"}`, phone, message)
	fmt.Println(dataStr)
	data := []byte(dataStr)
	req, err := http.NewRequest(method, urlLogin, bytes.NewBuffer(data))

	if err != nil {
		fmt.Println(err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("response Body:", string(body))
	return nil
}
