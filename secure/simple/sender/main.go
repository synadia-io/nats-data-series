package main

import (
	"fmt"
	"os"

	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

const Subject = "secure.data"

func main() {
	// get the public key of the receiver from the commandline arguments
	if len(os.Args) < 2 {
		fmt.Println("receiver public key is required")
		return
	}

	receiverPublicKey := os.Args[1]
	if err := run(receiverPublicKey); err != nil {
		fmt.Println(err)
	}
}

// run will launch the sender. The sender will send a message to the NATS server.
func run(receiverPublicKey string) error {
	// generate our encryption key
	xkey, err := nkeys.CreateCurveKeys()
	if err != nil {
		return fmt.Errorf("failed to generate xkey: %w", err)
	}

	// connect to the NATS server
	nc, err := natscontext.Connect("rethink")
	if err != nil {
		return fmt.Errorf("failed to connect to NATS server: %w", err)
	}
	defer nc.Close()

	if err := publishMessage(nc, xkey, receiverPublicKey); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func publishMessage(nc *nats.Conn, xkey nkeys.KeyPair, receiverPublicKey string) error {
	var err error
	msg := nats.NewMsg(Subject)

	// we will put the public key of the sender in the header of the message. This will allow the receiver to know who sent the message and to decrypt it.
	pk, err := xkey.PublicKey()
	if err != nil {
		return fmt.Errorf("failed to get the sender's public key: %w", err)
	}
	msg.Header.Add("sender", pk)

	// encrypt the message
	encrypted, err := xkey.Seal([]byte("Hello Secure World"), receiverPublicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	msg.Data = encrypted
	if err := nc.PublishMsg(msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	fmt.Printf(">> SENT %s from %s to %s\n", msg.Data, pk, receiverPublicKey)

	return nil
}
