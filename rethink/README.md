# Rethinking the system of record

My dad always told me “It is through mistakes we learn”. My answer, “True, if you can remember why it was a mistake”.
Reflection and the ability to recall information are crucial tools for us as humans, but they are also the foundations
on which an organization is built.

Organizations use databases to keep track of all the information they gather as part of their business. This information
is used to make decisions, and the quality of the decisions depends on the quality of the data. If the data is
inaccurate, the decisions made based on that data will be inaccurate as well. While many people think of data accuracy
as being something that expresses how up-to-date the data is, I think it goes a bit further. I am convinced the way data
is stored is just as important as the data itself.

Imagine you have a database that stores information about the stock of products in our warehouse. You have a table that
stores the location, product name and the quantity::

| Product Name | Location | Quantity |
|:-------------|:---------|:---------|
| Apples       | A43      | 5        |

The next time we take a look at the table, we see the following:

| Product Name | Location | Quantity |
|:-------------|:---------|:---------|
| Apples       | A43      | **2**    |

As you can see, the amount of apples in stock has changed. Without knowing anything more we may conclude that 3 apples
were bought which impacted the stock we hold. But was that really what happened? What if the stock was updated because 3
apples were rotten and had to be thrown away? Or if 3 apples have been moved to a different location or to a different
warehouse? The point is, without knowing the context in which the data was updated, we cannot be sure what the data
actually means.

## An alternative approach

Let's take a different approach. Instead of storing what has changed, we will store what actually happened. Instead of a
table, we will use a log. Every time something happens to the stock, we will write it down in the log. This way, we can
always go back and see what happened. Imagine the log will look something like this before we update the stock:

| Time                | Action    | Payload                                                   |
|:--------------------|:----------|:----------------------------------------------------------|
| 2020-01-01 12:23:00 | Corrected | { "Product": "Apples", "Location": "A43", "Quantity": 5 } |

Now let's do the same update as before, but this time we will add a new entry to the log:

| Time                | Action    | Payload                                                            |
|:--------------------|:----------|:-------------------------------------------------------------------|
| 2020-01-01 12:00:00 | Corrected | { "Product": "Apples", "Location": "A43", "Quantity": 5 }          |
| 2020-01-01 12:00:00 | Moved     | { "Product": "Apples", "From": "A43", "To": "G15", "Quantity": 3 } |

As you can see, we now have a much better understanding of what happened. We can see that 3 apples were moved from
location A43 to location G15. The idea is that instead of storing the result of an action (stock being 2), we store the
action itself. If by now you are thinking "hey that sounds like event-sourcing", then you are absolutely right. But as
you will notice further on, we are taking more of a pragmatic approach when it comes to the buzzword. Let's just call it
an event-driven architecture, shall we?

While this whole approach allows us to have a better understanding of what happened, there is a downside. The log is
optimized for writing but when it comes to reading, we must go through the entire log to find the information we are
looking for. The answer to this problem is simple: We don't use the log to perform ad hoc queries. Instead, we use the
logs as the data wires running through our organization. Anyone who needs to know what happened can tap into those logs
and start building their own views of the data based on the events they are interested in. This also means anyone can
interpret the data in their own way, allowing them to account for the particularities of their domain.

For example, the warehouse manager may be interested in knowing the stock of products in the warehouse. She can build a
view that listens to all the stock updates and keeps track of the stock in her own database. This might result in a
table that closely resembles the one we had in the beginning.

The marketing department might be interested in knowing which products are moving the fastest. They can build a view
that listens to all the stock updates and keeps track of the sales. This might result in a table that looks like this:

| Product Name | Sold Last Hour | Sold Last Day | Sold Last Week |
|:-------------|:---------------|:--------------|:---------------|
| Oranges      | 3              | 10            | 50             |
| Apples       | 2              | 5             | 20             |

The point is, everyone can build their own views of the data based on the events they are interested in without having
to deal with the bias introduced by others. They take the raw facts (events) and interpret them in their own way. This
is the power of event-driven architectures.

## What Is An Event?

Let's move a level deeper and see how we can make this work in practice. We will use NATS as the foundation for our
approach since it offers a lot of features that make it easy to implement event-driven architectures. We'll discuss
those as we go along.

But first let's start by looking at how we can model the events. Roughly, an event is made of the following parts:

- Metadata about the event, which we call the `headers`
- The actual data of the event, which we call the `payload`

### Headers

The headers are just a set of key/value pairs we can use to store information about the event. While this allows us to
store pretty much anything in the headers, we want to keep the headers as small as possible. Headers can be decoded
separately from the payload, which means we can use them to filter events without having to decode the payload. It makes
sense to prefix your header keys with an identifier unique to your organization to avoid conflicts with other systems.
In our examples we will use `myorg` as the prefix.

I suggest to at least include the following headers in every event:

- `myorg_format`: the mimetype of the payload. (e.g. application/json)
- `myorg_type`: the type of the event. (e.g. stock.moved)
- `myorg_msg_id`: a unique identifier for the event. (e.g. a UUID)
- `myorg_timestamp_ms`: the time the event was created. (e.g. 1580000000000\)

Note that we use milliseconds since the epoch for the timestamp. This makes it easy to work with timestamps without the
need of parsing them. The message format we will use is JSON, so the headers will look something like this:

```
{
  "myorg_format": "application/json",
  "myorg_type": "stock.moved",
  "myorg_msg_id": "a1b2c3d4",
  "myorg_timestamp_ms": 1580000000000
}
```

### Payload

As an event describes something that happened, the payload provides the information needed for other systems to
interpret the event. From NATS’ point of view, the payload is just a sequence of bytes, but we would like to put a bit
more structure into it. There are two trains of thought on what that structure should look like.

- Payload as arguments
- Payload as changes

#### Payload as arguments

The first one is to think of the payload as the arguments passed to a function. For the stock moved event, it isn't hard
to imagine a `move_stock(product, from, to quantity)` function that would have created this event. The payload would
then look like this:

```
{
  "product_name": "Apples",
  "from": "A43",
  "to": "G15",
  "quantity": 3
}
```

The benefit of this approach is that it is easy to understand what the event is about. The downside is that whoever
consumes these events needs to know about the particularities of the operation that happened. They need to know what the
move\_stock function actually does. This is not a problem if you control all the systems that consume the events, but it
can be a problem if you want to share the events with others.

#### Payload as changes

The second one is to take a more generic approach and think of the payload as a sequence of changes that were made as a
result of the event. In this case, the payload would look like this:

```
[
  {"op": "add", "path": "/location/A43/quantity", "value": -3},
  {"op": "add", "path": "/location/G15/quantity", "value": 3}
]
```

The benefit of this approach is that whoever consumes these events shouldn't know about the particularities of the
operation that happened. They can just apply the changes to their data and be done with it. They don't need to know what
the move\_stock function actually does. We can clearly see what happened by looking at the changes.

I was planning to go deeper into the details of how to implement this
using [Conflict free replicated data types](https://en.wikipedia.org/wiki/Conflict-free_replicated_data_type)
and [CRDT-Patch](https://jsonjoy.com/specs/json-crdt-patch), but I think I will leave that for another time. Let me know
if you would be interested in that.

### Publishing Events

For the time being, let's take the first approach and put it all together. Below is an example of what a stock moved
event could look like:

```
header:
  "myorg_format": "application/json",
  "myorg_type": "stock.moved",
  "myorg_msg_id": "a1b2c3d4",
  "myorg_timestamp_ms": 1580000000000

payload:
  {
	"product_name": "Apples",
	"from": "A43",
	"to": "G15",
	"quantity": 3
  }
```

NATS offers us a way to publish these events to a subject. There can be millions of subjects in a NATS system, and you
can structure them in a tree-like way:

```
warehouse.43.product.Apples
```

Since NATS also allow us to subscribe to subjects using wildcards, we could subscribe to `warehouse.*.product.*` to
receive all events related to products in the warehouse, but we will get to that in a minute. There are some questions
from the audience that I would like to address first.

"Are events stored?"

Excellent question\! While core NATS does not store events, NATS offers a feature called JetStream that allows us to
store messages in a stream. This means we can store the events we publish and replay them later if needed. This comes in
handy when we want to use a stream as a buffer for operational purposes. We can easily restart consumers of that stream
without risking losing any events. But we can also use JetStream to store events for a longer period of time, allowing
us to re-process them if \- for example \- somebody discovers a bug in the processing of the events.

How cool is that\! Ok, take a moment to cool down from the excitement before we jump into the next section.

## Interpreting Events

So we know what an event looks like, but how do we interpret it? Let's play the role of the procurement department
responsible for ordering new stock. We are interested in knowing when stock is moved, sold or destroyed so we can update
our records and order new stock if needed.

### Updating our knowledge

We already know that those events will be published to the NATS system, so we just need to subscribe to the right
subjects. We can then write a small program that listens to those events and updates our records accordingly:

```go
func subscribeToNats(nc *nats.Conn) {
    // get access to JetStream
    js, err := jetstream.New(nc)
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
        case "stock.replenished":
            fmt.Printf("Received stock replenished event: %s\n", msg.Data())
        }
    
        msg.Ack()
    })
    if err != nil {
        panic(err)
    }
}
```

I know, I know, this is far from production level code, it merely serves as an example. But you get the idea. We
subscribe to the events we are interested in and update our records accordingly. But where do we store those records? We
could use a database for that, something like Redis or Postgres. But why not use NATS itself? Remember how I said NATS
offers a feature called JetStream that allows us to store messages in a stream? Well, we can use that to store our
records. Even better, JetStream already offers a Key/Value store that we can use to store our records. I was thrilled
when I found out about this feature, and I think it is a game changer. But let's not get ahead of ourselves.

Let's get access to that Key/Value store before we subscribe to the events. That way we can pass it to the functions
that handle the events:

```go
func subscribeToNats(nc *nats.Conn) {
	...

	kv, err := js.KeyValue(context.Background(), "RETHINK_STOCK")
	if err != nil {
		panic(err)
	}

	...

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
```

Time for the fun part. Let's implement the `handleStockSold` function that will update our records when stock is sold.
We will get the current stock from the Key/Value store, update it and store it back in the Key/Value store. But before
we get to that, we need to decode the payload into something we can work with. Let's start by defining a struct that
represents the payload:

```
type StockSoldEvent struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}
```

Now we can update our program to decode the payload into this struct, get the current stock from the Key/Value store,
update it and store it back in the Key/Value store.

```
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
```

And that's it\! We now have a program that listens to events and updates our records accordingly. We can run this
program as a service and it will keep our records up-to-date.

### Acting on our knowledge

Now we only need to send out an event when stock is running low. While we could do this in the same program by adding a
check to the `handleStockSold` function, it will quickly become a mess if we add more checks. Like what would happen if
one of these failed? Instead, I prefer to keep my programs focussed on one thing and one thing only. So let's create a
new program that listens to the stock and sends out an event when stock is running low.

That low stock event would look something like this:

```
type LowStockEvent struct {
	ProductName string `json:"product_name"`
}
```

Now we only want to send out this event **after** we have updated our records. But is this another application? How do
we know when the records have been updated? If only there was a way to listen to changes in the Key/Value store. Oh hang
on a second ...

Since KeyValue stores are built on top of messages running through JetStream, we can actually watch for changes to the
keys in the store\!

```go
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
```

Now the only thing left to do is to implement the `publishLowStockEvent` function that sends out the low stock event:

```go
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
```

There we go, we now have a program that listens to changes in the stock and sends out an event when stock is running
low.

## Conclusion

Time to wrap things up\! We have seen how we can use event-driven architectures to build systems in a more flexible way.
By using events to describe what happened, we can build systems that are easier to understand and maintain. We can build
views of the data based on the events we are interested in, without having to deal with the bias introduced by others.

We also saw that not only does NATS offer a great foundation for building event-driven architectures, but it also offers
features that make it easy to build these systems. We can use JetStream to store messages and Key/Value stores to store
our records and watch for changes.

Obviously, all of this merely scratches the surface of what is possible. Stay tuned for more articles on how to build
the next generation of organizations\!

If you want to experience the code in action, you can find it [here](http://github.com/calmera/nats-data-series)

I hope you enjoyed this article as much as I enjoyed writing it. But it doesn't stop there. Hop into the 
[Data Series Slack Channel](https://natsio.slack.com/archives/C081AMW4A48) and share your thoughts. I would love
to hear from you.

Until next time\!