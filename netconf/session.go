// Go NETCONF Client
//
// Copyright (c) 2013-2018, Juniper Networks, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netconf

import (
	"encoding/xml"
	"fmt"
	"github.com/adetalhouet/go-netconf/netconf/message"
	"strings"
)

// DefaultCapabilities sets the default capabilities of the client library
var DefaultCapabilities = []string{
	message.NetconfVersion10,
	message.NetconfVersion11,
}

// Session represents a NETCONF sessions with a remote NETCONF server
type Session struct {
	Transport    Transport
	SessionID    int
	Capabilities []string
	IsClosed     bool
	Listener     *Dispatcher
}

// NewSession creates a new NETCONF session using the provided transport layer.
func NewSession(t Transport) *Session {
	s := new(Session)
	s.Transport = t

	// Receive server Hello message
	serverHello, _ := s.ReceiveHello()
	s.SessionID = serverHello.SessionID
	s.Capabilities = serverHello.Capabilities

	s.Listener = &Dispatcher{}
	s.Listener.init()

	return s
}

// SendHello send the initial message through NETCONF to advertise supported capability.
func (session *Session) SendHello(hello *message.Hello) error {
	val, err := xml.Marshal(hello)
	if err != nil {
		return err
	}

	header := []byte(xml.Header)
	val = append(header, val...)
	err = session.Transport.Send(val)

	// Set Transport version after sending hello-message,
	// so the hello-message is sent using netconf:1.0 framing
	session.Transport.SetVersion("v1.0")
	for _, capability := range session.Capabilities {
		if strings.Contains(capability, message.NetconfVersion11) {
			session.Transport.SetVersion("v1.1")
			break
		}
	}

	// FIXME shouldn't be in SendHello function
	// Once the hello-message exchange is done, start listening to incoming messages
	session.listen()

	return err
}

// ReceiveHello is the first message received when connecting to a NETCONF server.
// It provides the supported capabilities of the server.
func (session *Session) ReceiveHello() (*message.Hello, error) {
	session.IsClosed = false

	hello := new(message.Hello)

	val, err := session.Transport.Receive()
	if err != nil {
		return hello, err
	}

	err = xml.Unmarshal(val, hello)
	return hello, err
}

// Close is used to close and end a session
func (session *Session) Close() error {
	session.IsClosed = true
	return session.Transport.Close()
}

// Listen starts a goroutine that listen to incoming messages and dispatch them as then are processed.
func (session *Session) listen() {
	go func() {
		for {
			rawXML, err := session.Transport.Receive()
			if err != nil {
				println(fmt.Errorf("failed to receive message %s", err))
				continue
			}
			var rawReply = string(rawXML)
			if strings.Contains(rawReply, "<rpc-reply") {

				rpcReply, err := message.NewRPCReply(rawXML)
				if err != nil {
					println(fmt.Errorf("failed to marshall message into an RPCReply. %s", err))
					continue
				}
				session.Listener.Dispatch(rpcReply.MessageID, 0, rpcReply)

			} else if strings.Contains(rawReply, "<notification") {
				notification, err := message.NewNotification(rawXML)
				if err != nil {
					println(fmt.Errorf("failed to marshall message into an Notification. %s", err))
					continue
				}
				session.Listener.Dispatch(notification.SubscriptionID, 1, notification)
			} else {
				println(fmt.Errorf(fmt.Sprintf("unknown received message: \n%s", rawXML)))
			}

			if session.IsClosed {
				println("exist receiving loop")
				break
			}
		}
	}()
}

// DefaultLogRpcReplyCallback defines a default callback function
func (session *Session) DefaultLogRpcReplyCallback(eventId string) Callback {
	return func(event Event) {
		session.Listener.Remove(eventId)
		reply := event.RPCReply()
		if reply == nil {
			println("Failed to execute RPC")
		}
		if event.EventID() == eventId {
			println("Successfully executed RPC")
			println(reply.RawReply)
		}
	}
}

// DefaultLogNotificationCallback defines a default callback function
func (session *Session) DefaultLogNotificationCallback(eventId string) Callback {
	return func(event Event) {
		session.Listener.Remove(eventId)
		reply := event.Notification()
		if reply == nil {
			println("Failed to execute Notification")
		}
		if event.EventID() == eventId {
			println("Successfully executed Notification")
			println(reply.RawReply)
		}
	}
}
