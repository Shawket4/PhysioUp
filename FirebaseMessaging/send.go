package FirebaseMessaging

import (
	"PhysioUp/Models"
	"context"
	"log"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

var (
	app              *firebase.App
	messagingClient  *messaging.Client
	serviceAccountID string
)

func Setup() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get service account file path from environment
	serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
	if serviceAccountPath == "" {
		log.Println("FIREBASE_SERVICE_ACCOUNT_PATH not set, will use application default credentials")
	}

	// Initialize Firebase app
	ctx := context.Background()
	var err error

	if serviceAccountPath != "" {
		// Initialize with service account if provided
		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err = firebase.NewApp(ctx, nil, opt)
	} else {
		// Use application default credentials
		app, err = firebase.NewApp(ctx, nil)
	}

	if err != nil {
		log.Fatalf("Failed to initialize Firebase app: %v", err)
	}

	// Initialize messaging client
	messagingClient, err = app.Messaging(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase messaging client: %v", err)
	}

	log.Println("Firebase messaging client initialized successfully")
}

func SendMessage(req Models.NotificationRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
	}

	message.Android = &messaging.AndroidConfig{
		Priority: "high",
		Notification: &messaging.AndroidNotification{
			Sound:    "default",
			Priority: messaging.PriorityHigh,
		},
	}

	// Add APNS (iOS) config
	message.APNS = &messaging.APNSConfig{
		Headers: map[string]string{
			"apns-priority": "10",
		},
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Alert: &messaging.ApsAlert{
					Title: req.Title,
					Body:  req.Body,
				},
				Sound: "default",
			},
		},
	}

	switch {
	case len(req.Tokens) == 1:
		message.Token = req.Tokens[0]
		_, err := messagingClient.Send(ctx, message)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			return err
		}
	case len(req.Tokens) > 1:

		// Send to multiple devices

		messages := make([]*messaging.Message, len(req.Tokens))
		for i, token := range req.Tokens {
			// Create a copy of the message for each token
			messageCopy := *message
			messageCopy.Token = token
			messages[i] = &messageCopy
		}

		_, err := messagingClient.SendEachForMulticast(ctx, &messaging.MulticastMessage{
			Tokens:       req.Tokens,
			Notification: message.Notification,
			Data:         message.Data,
			Android:      message.Android,
			APNS:         message.APNS,
		})
		if err != nil {
			log.Printf("Error sending multicast message: %v", err)
			return err
		}
	}
	return nil

}
