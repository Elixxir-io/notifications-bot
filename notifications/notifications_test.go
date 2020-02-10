////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package notifications

import (
	"firebase.google.com/go/messaging"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/notifications-bot/firebase"
	"gitlab.com/elixxir/notifications-bot/storage"
	"gitlab.com/elixxir/notifications-bot/testutil"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"os"
	"strings"
	"testing"
	"time"
)

// Basic test to cover RunNotificationLoop, including error sending
func TestRunNotificationLoop(t *testing.T) {
	impl := getNewImpl()
	impl.pollFunc = func(*Impl) (i []string, e error) {
		return []string{"test1", "test2"}, nil
	}
	impl.notifyFunc = func(fcm *messaging.Client, s3 string, comm *firebase.FirebaseComm, storage storage.Storage) (s string, e error) {
		return "good", nil
	}
	killChan := make(chan struct{})
	errChan := make(chan error)
	go func() {
		time.Sleep(10 * time.Second)
		killChan <- struct{}{}
	}()
	impl.RunNotificationLoop(3, killChan, errChan)
}

// Test notificationbot's notifyuser function
// this mocks the setup and send functions, and only tests the core logic of this function
func TestNotifyUser(t *testing.T) {
	badsend := func(firebase.FBSender, string) (string, error) {
		return "", errors.New("Failed")
	}
	send := func(firebase.FBSender, string) (string, error) {
		return "", nil
	}
	fc_badsend := firebase.NewMockFirebaseComm(t, badsend)
	fc := firebase.NewMockFirebaseComm(t, send)

	_, err := notifyUser(nil, "test", fc_badsend, testutil.MockStorage{})
	if err == nil {
		t.Errorf("Should have returned an error")
	}

	_, err = notifyUser(nil, "test", fc, testutil.MockStorage{})
	if err != nil {
		t.Errorf("Failed to notify user properly")
	}
}

// Unit test for startnotifications
// tests logic including error cases
func TestStartNotifications(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("Failed to get working dir: %+v", err)
		return
	}

	params := Params{
		Address:       "0.0.0.0:42010",
		PublicAddress: "0.0.0.0:42010",
	}

	n, err := StartNotifications(params, false, true)
	if err == nil || !strings.Contains(err.Error(), "failed to read key at") {
		t.Errorf("Should have thrown an error for no key path")
	}

	params.KeyPath = wd + "/../testutil/badkey"
	n, err = StartNotifications(params, false, true)
	if err == nil || !strings.Contains(err.Error(), "Failed to parse notifications server key") {
		t.Errorf("Should have thrown an error bad key")
	}

	params.KeyPath = wd + "/../testutil/cmix.rip.key"
	n, err = StartNotifications(params, false, true)
	if err == nil || !strings.Contains(err.Error(), "failed to read certificate at") {
		t.Errorf("Should have thrown an error for no cert path")
	}

	params.CertPath = wd + "/../testutil/badkey"
	n, err = StartNotifications(params, false, true)
	if err == nil || !strings.Contains(err.Error(), "Failed to parse notifications server cert") {
		t.Errorf("Should have thrown an error for bad certificate")
	}

	params.CertPath = wd + "/../testutil/cmix.rip.crt"
	n, err = StartNotifications(params, false, true)
	if err != nil {
		t.Errorf("Failed to start notifications successfully: %+v", err)
	}
	if n.notificationKey == nil {
		t.Error("Did not set key")
	}
	if n.notificationCert == nil {
		t.Errorf("Did not set cert")
	}
}

// unit test for newimplementation
// tests logic and error cases
func TestNewImplementation(t *testing.T) {
	instance := getNewImpl()

	impl := NewImplementation(instance)
	if impl.Functions.RegisterForNotifications == nil || impl.Functions.UnregisterForNotifications == nil {
		t.Errorf("Functions were not properly set")
	}
}

// Dummy comms to unit test pollfornotifications
type mockPollComm struct{}

func (m mockPollComm) RequestNotifications(host *connect.Host) (*pb.IDList, error) {
	return &pb.IDList{
		IDs: []string{"test"},
	}, nil
}
func (m mockPollComm) GetHost(hostId string) (*connect.Host, bool) {
	return &connect.Host{}, true
}
func (m mockPollComm) AddHost(id, address string, cert []byte, disableTimeout, enableAuth bool) (host *connect.Host, err error) {
	return nil, nil
}
func (m mockPollComm) RequestNdf(host *connect.Host, message *pb.NDFHash) (*pb.NDF, error) {
	return nil, nil
}

type mockPollErrComm struct{}

func (m mockPollErrComm) RequestNotifications(host *connect.Host) (*pb.IDList, error) {
	return nil, errors.New("failed to poll")
}
func (m mockPollErrComm) GetHost(hostId string) (*connect.Host, bool) {
	return nil, false
}
func (m mockPollErrComm) AddHost(id, address string, cert []byte, disableTimeout, enableAuth bool) (host *connect.Host, err error) {
	return nil, nil
}
func (m mockPollErrComm) RequestNdf(host *connect.Host, message *pb.NDFHash) (*pb.NDF, error) {
	return nil, nil
}

// Unit test for PollForNotifications
func TestPollForNotifications(t *testing.T) {
	impl := &Impl{
		Comms: mockPollComm{},
	}
	errImpl := &Impl{
		Comms: mockPollErrComm{},
	}
	_, err := pollForNotifications(errImpl)
	if err == nil {
		t.Errorf("Failed to poll for notifications: %+v", err)
	}

	_, err = pollForNotifications(impl)
	if err != nil {
		t.Errorf("Failed to poll for notifications: %+v", err)
	}
}

// Unit test for RegisterForNotifications
func TestImpl_RegisterForNotifications(t *testing.T) {
	impl := getNewImpl()
	impl.Storage = testutil.MockStorage{}
	wd, _ := os.Getwd()
	crt, _ := utils.ReadFile(wd + "/../testutil/cmix.rip.crt")
	host, err := connect.NewHost("test", "0.0.0.0:420", crt, false, false)
	if err != nil {
		t.Errorf("Failed to create dummy host: %+v", err)
	}
	err = impl.RegisterForNotifications([]byte("token"), &connect.Auth{
		IsAuthenticated: true,
		Sender:          host,
	})
	if err != nil {
		t.Errorf("Failed to register for notifications: %+v", err)
	}
}

// Unit test that tests to see that updateNDF will in fact update the ndf object inside of IMPL
func TestImpl_UpdateNdf(t *testing.T) {
	impl := getNewImpl()
	testNdf, _, err := ndf.DecodeNDF(ExampleNdfJSON)
	if err != nil {
		t.Logf("%+v", err)
	}

	err = impl.UpdateNdf(testNdf)
	if err != nil {
		t.Errorf("Failed to update ndf")
	}

	if impl.ndf != testNdf {
		t.Logf("Failed to change ndf")
		t.Fail()
	}
}

// Unit test for UnregisterForNotifications
func TestImpl_UnregisterForNotifications(t *testing.T) {
	impl := getNewImpl()
	impl.Storage = testutil.MockStorage{}
	wd, _ := os.Getwd()
	crt, _ := utils.ReadFile(wd + "/../testutil/cmix.rip.crt")
	host, err := connect.NewHost("test", "0.0.0.0:420", crt, false, false)
	if err != nil {
		t.Errorf("Failed to create dummy host: %+v", err)
	}
	err = impl.UnregisterForNotifications(&connect.Auth{
		IsAuthenticated: true,
		Sender:          host,
	})
	if err != nil {
		t.Errorf("Failed to register for notifications: %+v", err)
	}
}

// func to get a quick new impl using test creds
func getNewImpl() *Impl {
	wd, _ := os.Getwd()
	params := Params{
		Address:       "0.0.0.0:4200",
		KeyPath:       wd + "/../testutil/cmix.rip.key",
		CertPath:      wd + "/../testutil/cmix.rip.crt",
		PublicAddress: "0.0.0.0:0",
		FBCreds:       "",
	}
	instance, _ := StartNotifications(params, false, true)
	return instance
}
