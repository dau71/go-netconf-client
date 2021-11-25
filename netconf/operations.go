/*
Copyright 2021. Alexis de Talhouët

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package netconf

import (
	"encoding/xml"
	"fmt"
	"github.com/adetalhouet/go-netconf/netconf/message"
	"time"
)

// CreateNotificationStream is a convenient method to create a notification stream registration.
// TODO limitation - for now, we can only register one stream per session, because when a notification is received
// there is no way to attribute it to a specific stream
func (session *Session) CreateNotificationStream(
	stopTime string, startTime string, stream string, callback Callback,
) error {
	if session.IsNotificationStreamCreated {
		return fmt.Errorf(
			"there is already an active notification stream subscription. " +
				"A session can only support one notification stream at the time",
		)
	}
	session.Listener.Register(message.NetconfNotificationStreamHandler, callback)
	sub := message.NewCreateSubscription(stopTime, startTime, stream)
	rpc, err := session.SyncRPC(sub)
	if err != nil || len(rpc.Errors) != 0 {
		return fmt.Errorf("fail to create notification stream with errors: %s. Error: %s", rpc.Errors, err)
	}
	session.IsNotificationStreamCreated = true
	return nil
}

// AsyncRPC is used to send an RPC method and receive the response asynchronously.
func (session *Session) AsyncRPC(operation message.RPCMethod, callback Callback) error {

	// get XML payload
	request, err := marshall(operation)
	if err != nil {
		return err
	}

	// register the listener for the message
	session.Listener.Register(operation.GetMessageID(), callback)

	fmt.Println(fmt.Sprintf("\nSending RPC"))
	err = session.Transport.Send(request)
	if err != nil {
		return err
	}

	return nil
}

// SyncRPC is used to execute an RPC method and receive the response synchronously
func (session *Session) SyncRPC(operation message.RPCMethod) (*message.RPCReply, error) {

	// get XML payload
	request, err := marshall(operation)
	if err != nil {
		return nil, err
	}

	// setup and register callback
	var reply = message.RPCReply{}
	var replyReceived = false
	callback := func(event Event) {
		reply = *event.RPCReply()
		replyReceived = true
		println("Successfully executed RPC")
		println(reply.RawReply)
	}
	session.Listener.Register(operation.GetMessageID(), callback)

	// send rpc
	fmt.Println(fmt.Sprintf("\n\nSending RPC"))
	err = session.Transport.Send(request)
	if err != nil {
		return nil, err
	}

	// wait for reply
	// TODO add support for timeout
	for !replyReceived {
		time.Sleep(100 * time.Millisecond)
	}

	return &reply, nil
}

func marshall(operation interface{}) ([]byte, error) {
	request, err := xml.Marshal(operation)
	if err != nil {
		return nil, err
	}

	header := []byte(xml.Header)
	request = append(header, request...)
	return request, nil
}
