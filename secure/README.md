# Secure

Organizations are not the isolated monoliths they used to be. They are now a collection of services, each with its own data and responsibilities. These services need to communicate with each other, and they need to do so securely.

In this part of the series, we will explore why securing the channel just isn't enough, and how to secure the messages themselves.

Many organizations rely on TLS to secure their messages. TLS is a protocol that encrypts the channel between two parties and by doing so, no one can listen in on the messages being sent between the two parties. It is also a way to make sure that the messages are sent from the source we expect them to be from.

However, TLS **only** secures the channel. It doesn't secure the messages themselves. Anyone with access to the channel can still see the messages being sent. This is why it is important to secure the messages themselves, not just the channel.

## Signing
When you receive a message, how do you know it is from the source you expect it to be from? How do you know it hasn't been tampered with? We would need a way to validate whether the message is actually from a party we trust. This is where signing comes in.

Signing is a way to validate the authenticity of a message. It is a way to prove that the message is from the source we expect it to be from and that the message has not been tampered with. In general, it comes down to
adding a signature to a message, much like we put our signature on a contract. This signature is generated using a secret key that only the sender knows. The receiver only knows the public part of the sender's key,
but can validate whether the message was signed with a sender's secret key.

This way, the receiver can be sure the message has been encrypted with the sender's secret key, and that the message has not been tampered with. Obviously, anyone with the sender's secret key can sign a message, so it is important to keep the sender's secret key ... well ... secret.

## Encrypting
Signing is a way to validate the authenticity of a message, but it doesn't protect the message itself. Anyone with access to the message can still see its contents. Encrypting is a way to protect the message itself. It is a way to make sure that only the intended recipient can read the message.

When encrypting a message, the sender uses the recipient's public key to encrypt the message. The recipient can then use their private key to decrypt the message. This way, only the recipient can read the message. Anyone who intercepts the message will only see a garbled mess.

The `nkeys` library will allow us to create a curved key pair to encrypt our message with. When encrypting, we will need to pass the public key of the receiver. This way, only the receiver can decrypt the message with their private key.

The receiver in turn will need the sender's public key to decrypt the message. We will pass the public key of the sender along with the message, so the receiver can validate the message's origin.

This means we can use the same encryption key for both encryption and signing. How lovely!

## Receiver
Let's start with the receiver, shall we? The first thing we will need to do is generate the receiver's key pair. We will use the `nkeys` library to do this. The receiver will need to keep their private key secret, but they can share their public key with the sender.

Just for the sake of being brief, we will leave out some of the boilerplace code. Refer to the code in the `receiver` directory for the full example.

```go
xkey, err := nkeys.CreateCurveKeys()
if err != nil {
	return fmt.Errorf("failed to generate xkey: %w", err)
}

pubKey, err := xkey.PublicKey()
if err != nil {
	return fmt.Errorf("failed to get the receiver's public key: %w", err)
}
```

Once we have our keypair and our public key, we will connect to NATS and start listening for messages:

```go
// connect to the NATS server
nc, err := natscontext.Connect("rethink")
if err != nil {
	return fmt.Errorf("failed to connect to NATS server: %w", err)
}
defer nc.Close()

// subscribe to the secure.data subject
sub, err := nc.Subscribe(Subject, func(msg *nats.Msg) {
	// TODO: handle the message
})
if err != nil {
	return fmt.Errorf("failed to subscribe to the secure.data subject: %w", err)
}
defer sub.Unsubscribe()
```

See that TODO message? That's where we will handle the message. We will need to decrypt the message using our private key, and validate the message using the sender's public key. Let's do that now:

```go
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
```

And that's it! We have our receiver set up. Celebrations all around!

## Sender
Now that we have our receiver set up, let's move on to the sender. The sender will need to generate their key pair, and share their public key with the receiver. The sender will also need the receiver's public key to encrypt the message.

Just like with the receiver, we will leave out some of the boilerplate code. Refer to the code in the `sender` directory for the full example.

```go
xkey, err := nkeys.CreateCurveKeys()
if err != nil {
	return fmt.Errorf("failed to generate xkey: %w", err)
}
```

Once we have our keypair, we will connect to NATS and reach out to a function to send a message (I like to keep my code clean and organized). Take special notice of the `receiverPublicKey` variable. This is the public key of the receiver, which we will use to encrypt the message. It will need to be provided on the CLI or being discovered in some other way.

```go
nc, err := natscontext.Connect("rethink")
if err != nil {
	return fmt.Errorf("failed to connect to NATS server: %w", err)
}
defer nc.Close()

if err := publishMessage(nc, xkey, receiverPublicKey); err != nil {
	return fmt.Errorf("failed to publish message: %w", err)
}
```

The `publishMessage` function will take care of encrypting the message and sending it to the receiver:

```go
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
```

And that's our sender! We have our sender and receiver set up. Time to test our setup!

## Testing
To test our setup, we will need to run the receiver and sender. We will need to provide the receiver's public key to the sender, so the sender can encrypt the message. When the receiver
starts, it will print the public key it generated.

```bash
$ go run receiver/main.go
Receiver Public Key: XB2ME3SWDBSHZVZDGJR6AO47VFJ22EJNFTWGF6QFJ34UAIW65Q2QOI3S
Receiver is running, press Ctrl+C to stop
```

Copy the public key and provide it to the sender:

```bash
$ go run sender/main.go XB2ME3SWDBSHZVZDGJR6AO47VFJ22EJNFTWGF6QFJ34UAIW65Q2QOI3S
>> SENT xkv1|,�#L�;�(j'XR�xDcAb#M�'=�!�W from XBJCUSIMF6SOELLC3D7PKBR7LNBSE2S2FTA5BRK3T2DGU4YU5DSBKVC7 to XB2ME3SWDBSHZVZDGJR6AO47VFJ22EJNFTWGF6QFJ34UAIW65Q2QOI3S
```

The sender will quit once the message is sent, but if all went well, you should see the message being printed by the receiver:

```bash
Received message from XBJCUSIMF6SOELLC3D7PKBR7LNBSE2S2FTA5BRK3T2DGU4YU5DSBKVC7
The sender sent us "Hello Secure World"
```

So this shows we can actually read data that's sent to us. But what if we spawn another receiver? That new receiver will get a different public key, and won't be able to decrypt the message. The message was encrypted with the first receiver's public key, so only the first receiver can decrypt it.

```bash
 go run receiver/main.go
Receiver Public Key: XA7S7VZEEGE6TEBJFV4PO5DI7BIVBCZ4HA4LUDJ673RJZSEFQOHWB5C5
Receiver is running, press Ctrl+C to stop
Received message from XCAGOZTFPUN34K34NY5TBDIM2NJGWSZ2ZXQMVUG7AY3XVRWZTOGU73F3
Failed to decrypt message: nkeys: could not decrypt input
```

But there is a bit of a hidden elephant in the room. The sender needs to know the public key of the receiver. This is not always practical. What if we have multiple receivers? What if we don't know who the receiver is going to be? What if we want to use a pub/sub paradigm where multiple receivers need to be able to read the message from a single sender?

One approach could be to give all your receivers the same public key. This way, all receivers can decrypt the message. But this also means that all receivers can read the message. This is not always desirable. What if we want to send a message to multiple receivers, but only certain receivers should be able to read the message?

Stay with me for the second post in this series where we will explore how to use the principles shown here to give our system the ability to secure a single message for multiple receivers. We will even go one step further and allow a message to be sent to multiple receivers, but depending on the rights of the receiver, they will only be able to read certain parts of the message. This opens the door to do permission-based messaging, where only certain receivers can read certain parts of the message. 

For now, grab a cup of tea and enjoy the fact that you have secured your messages. Time for a well-deserved break!

