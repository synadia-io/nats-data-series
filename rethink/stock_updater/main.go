package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type StockSoldEvent struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

type StockReplenishedEvent struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

func main() {
	// Connect to the nats server created during the setup.
	nc, err := natscontext.Connect("rethink")
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	// Subscribe to all stock events
	subscribeToNats(nc)

	// Keep the program running
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Stock Updater is running, press Ctrl+C to stop")
	<-sigs
}

func subscribeToNats(nc *nats.Conn) {
	js, err := jetstream.New(nc)
	if err != nil {
		panic(err)
	}

	kv, err := js.KeyValue(context.Background(), "RETHINK_STOCK")
	if err != nil {
		panic(err)
	}

	// Create a consumer to listen to stock events
	c, err := js.CreateOrUpdateConsumer(context.Background(), "RETHINK_WAREHOUSE", jetstream.ConsumerConfig{
		Name:          "stock_updater",
		FilterSubject: "warehouse.*.product.*",
	})
	if err != nil {
		panic(err)
	}

	// Start consuming messages
	_, err = c.Consume(func(msg jetstream.Msg) {
		// get the type header
		eventType := msg.Headers().Get("myorg_type")

		switch eventType {
		case "stock.sold":
			fmt.Printf("Received stock sold event: %s\n", msg.Data())
			handleStockSold(kv, msg.Data())
		case "stock.replenished":
			fmt.Printf("Received stock replenished event: %s\n", msg.Data())
			handleStockReplenished(kv, msg.Data())
		}

		msg.Ack()
	})
	if err != nil {
		panic(err)
	}
}

func handleStockSold(kv jetstream.KeyValue, data []byte) {
	var evt StockSoldEvent
	json.Unmarshal(data, &evt)

	currentStock := 0
	currentEntry, _ := kv.Get(context.Background(), evt.ProductName)
	if currentEntry != nil {
		currentStock, _ = strconv.Atoi(string(currentEntry.Value()))
	}

	newStock := currentStock - evt.Quantity
	kv.Put(context.Background(), evt.ProductName, []byte(strconv.Itoa(newStock)))
}

func handleStockReplenished(kv jetstream.KeyValue, data []byte) {
	var evt StockReplenishedEvent
	json.Unmarshal(data, &evt)

	currentStock := 0
	currentEntry, _ := kv.Get(context.Background(), evt.ProductName)
	if currentEntry != nil {
		currentStock, _ = strconv.Atoi(string(currentEntry.Value()))
	}

	newStock := currentStock + evt.Quantity
	kv.Put(context.Background(), evt.ProductName, []byte(strconv.Itoa(newStock)))
}
