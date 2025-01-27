package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

const Subject = "secure.data"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	xkey, err := nkeys.CreateCurveKeys()
	if err != nil {
		return fmt.Errorf("failed to generate xkey: %w", err)
	}

	pubKey, err := xkey.PublicKey()
	if err != nil {
		return fmt.Errorf("failed to get the receiver's public key: %w", err)
	}

	// connect to the NATS server
	nc, err := natscontext.Connect("rethink")
	if err != nil {
		return fmt.Errorf("failed to connect to NATS server: %w", err)
	}
	defer nc.Close()

	// subscribe to the secure.data subject
	sub, err := nc.Subscribe(Subject, func(msg *nats.Msg) {
		// get the sender public key from the message
		sender := msg.Header.Get("sender")
		fmt.Printf("Received message from %s\n", sender)

		// decrypt the message
		decrypted, err := xkey.Open(msg.Data, sender)
		if err != nil {
			fmt.Printf("Failed to decrypt message: %s\n", err)
			return
		}

		fmt.Printf("The sender sent us %q\n", decrypted)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to the secure.data subject: %w", err)
	}
	defer sub.Unsubscribe()

	// Keep the program running
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	fmt.Printf("Receiver Public Key: %s\n", pubKey)
	fmt.Println("Receiver is running, press Ctrl+C to stop")
	<-sigs

	return nil
}
