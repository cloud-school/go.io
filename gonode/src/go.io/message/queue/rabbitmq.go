package message_queue
// This example declares a durable Exchange, and publishes a single message to
// that Exchange with a given routing key.
//

import (
    "flag"
    "fmt"
    "github.com/streadway/amqp"
    "log"
)

var (
    uri          = flag.String("uri", "amqp://guest:guest@localhost:5672/", "AMQP URI")
    exchange     = flag.String("exchange", "test-exchange", "Durable, non-auto-deleted AMQP exchange name")
    exchangeType = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
    queue        = flag.String("queue", "test-queue", "Ephemeral AMQP queue name")
    bindingKey   = flag.String("key", "test-key", "AMQP binding key")
    consumerTag  = flag.String("consumer-tag", "simple-consumer", "AMQP consumer tag (should not be blank)")
    // lifetime     = flag.Duration("lifetime", 0*time.Second, "lifetime of process before shutdown (0s=infinite)")
)

type MessageListener struct {
    consumer *Consumer
}

func GetConsumer() (*Consumer, error) {
    c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
    if err != nil {
        log.Fatalf("%s", err)
    }
    return c, err
}

type Consumer struct {
    conn    *amqp.Connection
    channel *amqp.Channel
    tag     string
    done    chan error
}

func NewConsumer(amqpURI, exchange, exchangeType, queue, key, ctag string) (*Consumer, error) {
    c := &Consumer{
        conn:    nil,
        channel: nil,
        tag:     ctag,
        done:    make(chan error),
    }

    var err error

    log.Printf("dialing %s", amqpURI)
    c.conn, err = amqp.Dial(amqpURI)
    if err != nil {
        return nil, fmt.Errorf("Dial: %s", err)
    }

    log.Printf("got Connection, getting Channel")
    c.channel, err = c.conn.Channel()
    if err != nil {
        return nil, fmt.Errorf("Channel: %s", err)
    }

    log.Printf("got Channel, declaring Exchange (%s)", exchange)
    if err = c.channel.ExchangeDeclare(
        exchange,     // name of the exchange
        exchangeType, // type
        true,         // durable
        false,        // delete when complete
        false,        // internal
        false,        // noWait
        nil,          // arguments
    ); err != nil {
        return nil, fmt.Errorf("Exchange Declare: %s", err)
    }

    log.Printf("declared Exchange, declaring Queue (%s)", queue)
    state, err := c.channel.QueueDeclare(
        queue, // name of the queue
        true,  // durable
        false, // delete when usused
        false, // exclusive
        false, // noWait
        nil,   // arguments
    )
    if err != nil {
        return nil, fmt.Errorf("Queue Declare: %s", err)
    }

    log.Printf("declared Queue (%d messages, %d consumers), binding to Exchange (key '%s')",
        state.Messages, state.Consumers, key)

    if err = c.channel.QueueBind(
        queue,    // name of the queue
        key,      // bindingKey
        exchange, // sourceExchange
        false,    // noWait
        nil,      // arguments
    ); err != nil {
        return nil, fmt.Errorf("Queue Bind: %s", err)
    }

    log.Printf("Queue bound to Exchange, starting Consume (consumer tag '%s')", c.tag)
    deliveries, err := c.channel.Consume(
        queue, // name
        c.tag, // consumerTag,
        false, // noAck
        false, // exclusive
        false, // noLocal
        false, // noWait
        nil,   // arguments
    )
    if err != nil {
        return nil, fmt.Errorf("Queue Consume: %s", err)
    }

    go handle(deliveries, c.done)

    return c, nil
}

func (c *Consumer) Shutdown() error {
    // will close() the deliveries channel
    if err := c.channel.Cancel(c.tag, true); err != nil {
        return fmt.Errorf("Consumer cancel failed: %s", err)
    }

    if err := c.conn.Close(); err != nil {
        return fmt.Errorf("AMQP connection close error: %s", err)
    }

    defer log.Printf("AMQP shutdown OK")

    // wait for handle() to exit
    return <-c.done
}

func handle(deliveries <-chan amqp.Delivery, done chan error) {
    for d := range deliveries {
        log.Printf(
            "got %dB delivery: [%v] %s",
            len(d.Body),
            d.DeliveryTag,
            d.Body,
        )
    }
    log.Printf("handle: deliveries channel closed")
    done <- nil
}
