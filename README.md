# NATS Data Series
This repository contains the sample code for the NATS Data Series blog posts. Each post has its own directory
which includes a Taskfile that can be used to run the code.

## Prerequisites
We rely on a few things to be installed on your system to run the code. The following is a list of tools we
expect to be available:

- [Task](https://taskfile.dev/#/installation) for running the code and making your life easier
- [NATS cli](https://github.com/nats-io/natscli) for running the server and interacting with it
- [Go](https://golang.org/doc/install) for building the code

Make sure you have these installed before running the code.

## Running the server
To run the server, you can use the following command:
```shell
task server
```

This will start a server on your system that will be used as part of the examples. It will also create a
nats-cli context you can use to interact with the server.

run `nats context select rethink` to select it

## Get In Touch
Hop into the [Data Series Slack Channel](https://natsio.slack.com/archives/C081AMW4A48) and share your thoughts. I 
would love

## Posts
### [Introduction](./introduction)
The first post in the series sets the stage for the rest of the series. It is a brief recap of how we got
to where we are today and why we need to rethink the way we build systems.

### [Rethinking the system of record](./rethink)
This post explores how to use NATS to build a system of record that can be used to store and retrieve data
from a distributed system.


#### Running the code
1. Start the server if it hasn't been started yet
2. Run `task rethink:setup` to build the code and create the JetStream stream and KV store
3. Run `task rethink:listen:low_stock` to start listening for low stock events
4. Run the next few commands in their own terminal windows
    1. Run `task rethink:run:low_stock_detector` to start detecting low stock items
    2. Run `task rethink:run:stock_updater` to start updating the stock levels on `stock.sold` and `stock.replenished` events
    3. Run `task rethink:run:stock_event_generator` to start generating stock events

#### What you should see
The `stock_event_generator` will send change events and print out the updates it made:
```
Replenished 53 Pears (54 left)
Replenished 14 Grapes (15 left)
Sold 27 Pears (27 left)
Sold 60 Oranges (8 left)
...
```

The `stock_updater` will update the stock based on the `stock.sold` and `stock.replenished` events:
```
Received stock sold event: {"product_name":"Grapes","quantity":1}
Received stock replenished event: {"product_name":"Grapes","quantity":63}
Received stock replenished event: {"product_name":"Oranges","quantity":34}
Received stock sold event: {"product_name":"Apples","quantity":4}
Received stock sold event: {"product_name":"Pears","quantity":4}
...
```

The `low_stock_detector` will look at the stock levels and send an event when the stock is low. **It will not 
output anything but send a message `stock.low` message**

The `listener` will listen for the `stock.low` events and print out the message it received:
```
[#182] Received on "product.Pears"
myorg_type: stock.low
myorg_format: application/json
myorg_msg_id: mBZvheMnM10QcgS9ZPCguH

{"product_name":"Pears"}


[#183] Received on "product.Pears"
myorg_format: application/json
myorg_msg_id: A3WBvfAGUqwQCnZPeHBU8x
myorg_type: stock.low

{"product_name":"Pears"}
```

#### Cleaning up
1. Each of the tasks can be stopped by pressing `ctrl+c`. Just keep the server running for the time being
2. Run `task rethink:teardown` to remove the JetStream stream and KV store
3. Stop the server by pressing `ctrl+c`
4. Run `task clean` to remove any lingering server files