////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package firebase

import (
	"encoding/base64"
	"firebase.google.com/go/messaging"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"testing"

	"golang.org/x/net/context"

	firebase "firebase.google.com/go"

	"google.golang.org/api/option"
)

// function types for use in notificationsbot struct
type SetupFunc func(string) (*messaging.Client, context.Context, error)
type SendFunc func(FBSender, string, *mixmessages.NotificationData) (string, error)

// FirebaseComm is a struct which holds the functions to setup the messaging app and sending notifications
// Using a struct in this manner allows us to properly unit test the NotifyUser function
type FirebaseComm struct {
	SendNotification SendFunc
}

// FBSender is an interface which matches the send function in the messaging app, allowing us to unit test sendNotification
type FBSender interface {
	Send(context.Context, *messaging.Message) (string, error)
}

// NewFirebaseComm create a *FirebaseComm object with the proper setup and send functions
func NewFirebaseComm() *FirebaseComm {
	return &FirebaseComm{
		SendNotification: sendNotification,
	}
}

// NewMockFirebaseComm FOR TESTING USE ONLY: create a *FirebaseComm object with mocked setup and send funcs
func NewMockFirebaseComm(t *testing.T, sendFunc SendFunc) *FirebaseComm {
	if t == nil {
		panic("This method should only be used in tests")
	}
	return &FirebaseComm{
		SendNotification: sendFunc,
	}
}

// SetupMessagingApp initializes communication with firebase
// It returns a messaging client, a context object and an error
func SetupMessagingApp(serviceKeyPath string) (*messaging.Client, error) {
	ctx := context.Background()
	opt := option.WithCredentialsFile(serviceKeyPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, errors.Errorf("Error initializing app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, errors.Errorf("Error getting Messaging app: %+v", err)
	}

	return client, nil
}

// sendNotification accepts a registration token and service account file
// It gets the proper infrastructure, then builds & sends a notification through the firebase admin API
// returns string, error (string is of dubious use, but is returned for the time being)
func sendNotification(app FBSender, token string, data *mixmessages.NotificationData) (string, error) {
	ctx := context.Background()
	message := &messaging.Message{
		Notification: nil, // This must remain nil for the data to go to android apps in background
		Data: map[string]string{
			"MessageHash":         base64.StdEncoding.EncodeToString(data.MessageHash),
			"IdentityFingerprint": base64.StdEncoding.EncodeToString(data.IdentityFP),
		},
		APNS: &messaging.APNSConfig{ // APNS is apple's native notification service, this is ios specific config
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					ContentAvailable: true,
				},
				CustomData: map[string]interface{}{
					"MessageHash":         base64.StdEncoding.EncodeToString(data.MessageHash),
					"IdentityFingerprint": base64.StdEncoding.EncodeToString(data.IdentityFP),
				},
			},
		},
		Token: token,
	}

	resp, err := app.Send(ctx, message)
	if err != nil {
		return resp, errors.Wrapf(err, "Failed to send notification.  Response: %+v", resp)
	}
	return resp, nil
}
