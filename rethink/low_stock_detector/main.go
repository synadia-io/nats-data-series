package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nuid"
	"strconv"
)

const stockThreshold = 10

type LowStockEvent struct {
	ProductName string `json:"product_name"`
}

func main() {
	// Connect to the nats server created during the setup.
	nc, err := natscontext.Connect("rethink")
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		panic(err)
	}

	kv, err := js.KeyValue(context.Background(), "RETHINK_STOCK")
	if err != nil {
		panic(err)
	}

	// Watch for low stock events
	watchKeyValue(kv, nc)
}

func watchKeyValue(kv jetstream.KeyValue, nc *nats.Conn) {
	watcher, _ := kv.WatchAll(context.Background(), jetstream.UpdatesOnly())
	for e := range watcher.Updates() {
		if e == nil {
			continue
		}

		if e.Operation() == jetstream.KeyValuePut {
			stock, _ := strconv.Atoi(string(e.Value()))
			if stock < stockThreshold {
				publishLowStockEvent(nc, e.Key())
			}
		}
	}
}

func publishLowStockEvent(nc *nats.Conn, productName string) {
	evt := LowStockEvent{ProductName: productName}
	data, _ := json.Marshal(evt)

	subject := fmt.Sprintf("product.%s", productName)
	msg := nats.NewMsg(subject)
	msg.Header.Add("myorg_type", "stock.low")
	msg.Header.Add("myorg_format", "application/json")
	msg.Header.Add("myorg_msg_id", nuid.New().Next())
	msg.Data = data

	nc.PublishMsg(msg)
}
