package main

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type StockSoldEvent struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

type StockReplenishedEvent struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

var stockedProducts = []string{
	"Apples",
	"Oranges",
	"Pears",
	"Grapes",
}

// We keep track of the product quantity just to make sure we can't send events with invalid quantities.
var productQty = map[string]int{}

func main() {
	// Connect to the local nats server
	// Connect to the nats server created during the setup.
	nc, err := natscontext.Connect("rethink")
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	// Send out a random product update every second
	ticker := time.NewTicker(time.Second)

	// Keep the program running
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Sales Generator is running, press Ctrl+C to stop")

	for {
		select {
		case <-ticker.C:
			tick(nc)
		case <-sigs:
			return
		}
	}
}

func tick(nc *nats.Conn) {
	// -- get a random product
	product := stockedProducts[rand.IntN(len(stockedProducts))]

	// -- decide if we want to sell or replenish the stock
	qty, fnd := productQty[product]
	if !fnd || qty <= 1 {
		// The quantity is a random number between 1 and 100
		qtyReplenished := 1 + rand.IntN(99)
		publishStockReplenishedEvent(nc, product, qtyReplenished)
		productQty[product] += qtyReplenished
		fmt.Printf("Replenished %d %s (%d left)\n", qtyReplenished, product, productQty[product])
	} else {
		// The quantity is a random number between 1 and the number of products
		qtySold := 1 + rand.IntN(qty-1)
		publishStockSoldEvent(nc, product, qtySold)
		productQty[product] -= qtySold
		fmt.Printf("Sold %d %s (%d left)\n", qtySold, product, productQty[product])
	}
}

func publishStockSoldEvent(nc *nats.Conn, product string, qty int) {
	evt := StockSoldEvent{ProductName: product, Quantity: qty}
	data, _ := json.Marshal(evt)

	subject := fmt.Sprintf("warehouse.46.product.%s", product)
	msg := nats.NewMsg(subject)
	msg.Header.Add("myorg_type", "stock.sold")
	msg.Header.Add("myorg_format", "application/json")
	msg.Header.Add("myorg_msg_id", nuid.New().Next())
	msg.Data = data

	nc.PublishMsg(msg)
}

func publishStockReplenishedEvent(nc *nats.Conn, product string, qty int) {
	evt := StockReplenishedEvent{ProductName: product, Quantity: qty}
	data, _ := json.Marshal(evt)

	subject := fmt.Sprintf("warehouse.46.product.%s", product)
	msg := nats.NewMsg(subject)
	msg.Header.Add("myorg_type", "stock.replenished")
	msg.Header.Add("myorg_format", "application/json")
	msg.Header.Add("myorg_msg_id", nuid.New().Next())
	msg.Data = data

	nc.PublishMsg(msg)
}
