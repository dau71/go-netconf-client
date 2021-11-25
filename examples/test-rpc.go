package main

import (
	"fmt"
	"github.com/adetalhouet/go-netconf/netconf"
	"github.com/adetalhouet/go-netconf/netconf/message"
	"golang.org/x/crypto/ssh"
	"log"
	"time"
)

func main() {

	// java -jar lighty-notifications-device-15.0.1-SNAPSHOT.jar 12345
	testNotification()

	// java -jar lighty-toaster-multiple-devices-15.0.1-SNAPSHOT.jar --starting-port 20000 --device-count 200 --thread-pool-size 200
	//testRPC()
}

func testNotification() {

	notificationSession := createSession(12345)

	callback := func(event netconf.Event) {
		reply := event.Notification()
		println(reply.RawReply)
	}
	notificationSession.CreateNotificationStream("", "", "", callback)

	triggerNotification := "    <triggerDataNotification xmlns=\"yang:lighty:test:notifications\">\n        <ClientId>0</ClientId>\n        <Count>5</Count>\n        <Delay>1</Delay>\n        <Payload>just simple notification</Payload>\n    </triggerDataNotification>"
	rpc := message.NewRPC(triggerNotification)
	notificationSession.SyncRPC(rpc)

	err := notificationSession.CreateNotificationStream("", "", "", callback)
	if err == nil {
		panic("must fail")
	}

	d := message.NewCloseSession()
	notificationSession.AsyncRPC(d, defaultLogRpcReplyCallback(d.MessageID))

	notificationSession.Listener.Remove(message.NetconfNotificationStreamHandler)
	notificationSession.Listener.WaitForMessages()

	notificationSession.Close()
}

func testRPC() {
	for i := 0; i < 200; i++ {
		i := i
		go func() {
			number := 20000 + i
			session := createSession(number)
			defer session.Close()
			execRPC(session)
		}()
	}
}

// Execute all types of RPC against the device
// Add a 100ms delay after each RPC to leave enough time for the device to reply
// Else, too many request and things get bad.
func execRPC(session *netconf.Session) {

	// Get Config
	g := message.NewGetConfig(message.DatastoreRunning, message.FilterTypeSubtree, "")
	session.AsyncRPC(g, defaultLogRpcReplyCallback(g.MessageID))
	time.Sleep(100 * time.Millisecond)

	// Get
	gt := message.NewGet("", "")
	session.AsyncRPC(gt, defaultLogRpcReplyCallback(gt.MessageID))
	time.Sleep(100 * time.Millisecond)

	// Lock
	l := message.NewLock(message.DatastoreCandidate)
	session.AsyncRPC(l, defaultLogRpcReplyCallback(l.MessageID))
	time.Sleep(100 * time.Millisecond)

	// EditConfig
	data := "<toaster xmlns=\"http://netconfcentral.org/ns/toaster\">\n    <darknessFactor>750</darknessFactor>\n</toaster>"
	e := message.NewEditConfig(message.DatastoreCandidate, message.DefaultOperationTypeMerge, data)
	session.AsyncRPC(e, defaultLogRpcReplyCallback(e.MessageID))
	time.Sleep(100 * time.Millisecond)

	// Commit
	c := message.NewCommit()
	session.AsyncRPC(c, defaultLogRpcReplyCallback(c.MessageID))
	time.Sleep(100 * time.Millisecond)

	// Unlock
	u := message.NewUnlock(message.DatastoreCandidate)
	session.AsyncRPC(u, defaultLogRpcReplyCallback(u.MessageID))
	time.Sleep(100 * time.Millisecond)

	// RPC
	d := "    <make-toast xmlns=\"http://netconfcentral.org/ns/toaster\">\n        <toasterDoneness>9</toasterDoneness>\n        <toasterToastType>frozen-waffle</toasterToastType>\n     </make-toast>"
	rpc := message.NewRPC(d)
	session.AsyncRPC(rpc, defaultLogRpcReplyCallback(rpc.MessageID))
	time.Sleep(100 * time.Millisecond)

	// RPCs
	rpc2 := message.NewRPC(d)
	session.SyncRPC(rpc2)
	rpc3 := message.NewRPC(d)
	session.SyncRPC(rpc3)
	rpc4 := message.NewRPC(d)
	session.SyncRPC(rpc4)

	// Close Session
	d2 := message.NewCloseSession()
	session.AsyncRPC(d2, defaultLogRpcReplyCallback(d2.MessageID))

	session.Listener.WaitForMessages()
}

func createSession(port int) *netconf.Session {
	sshConfig := &ssh.ClientConfig{
		User:            "admin",
		Auth:            []ssh.AuthMethod{ssh.Password("admin")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := netconf.DialSSH(fmt.Sprintf("127.0.0.1:%d", port), sshConfig)
	if err != nil {
		log.Fatal(err)
	}
	capabilities := netconf.DefaultCapabilities
	err = s.SendHello(&message.Hello{Capabilities: capabilities})
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func defaultLogRpcReplyCallback(eventId string) netconf.Callback {
	return func(event netconf.Event) {
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
