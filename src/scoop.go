package main

import (
    "log"
    "flag"
    "fmt"
    "github.com/streadway/amqp"
)

var (
    username         = flag.String("username", "guest", "Username")
    password         = flag.String("password", "guest", "Password")
    hostname         = flag.String("hostname", "localhost", "Hostname")
    port             = flag.String("port", "5672", "Port")
    fromQueueName    = flag.String("from", "", "The queue name to consume messages from")
    toQueueName      = flag.String("to", "", "The queue name to deliver messages to")
    exchange         = flag.String("exchange", "", "The exchange name to deliver messages through")
    messageCount     = flag.Int("count", 1, "The number of messages to move between queues")
    verbose          = flag.Bool("v", false, "Turn on verbose mode")
    veryVerbose      = flag.Bool("vv", false, "Turn on very verbose mode")
    extremelyVerbose = flag.Bool("vvv", false, "Turn on extremely verbose mode")
)

func init() {
    flag.Parse()
}

func main() {

    if (*extremelyVerbose) {
        log.Printf("Extremely verbose mode enabled")
    } else if (*veryVerbose) {
        log.Printf("Very verbose mode enabled")
    } else if (*verbose) {
        log.Printf("Verbose mode enabled")
    }

    // Set the verbose modes accordingly
    if (*extremelyVerbose) {
        *veryVerbose = true
        *verbose = true
    } else if (*veryVerbose) {
       *verbose = true
    }

    if (*fromQueueName == "") {
        log.Printf("The from argument must be defined")
    }

    if (*toQueueName == "") {
        log.Printf("The to argument must be defined")
    }

    conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", *username, *password, *hostname, *port))
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    if (*verbose) {
        log.Printf("Moving %d messages from queue %s to %s", *messageCount, *fromQueueName, *toQueueName)
    }

    // Check if the queue exists, otherwise, fail
    // To do this, add the additional argument of passive to true: https://github.com/streadway/amqp/blob/master/channel.go#L758
    // so if the queue does exist, the command fails (but we need the latest code for that)

    fromQueue, err := ch.QueueDeclare(
        *fromQueueName, // name
        false,          // durable
        false,          // delete when unused
        false,          // exclusive
        false,          // no-wait
        nil,            // arguments
    )
    failOnError(err, "Failed to declare a queue")

    if (*veryVerbose) {
        log.Printf("There are %d messages in queue %s", fromQueue.Messages, fromQueue.Name)
    }

    toQueue, err := ch.QueueDeclare(
        *toQueueName, // name
        false,        // durable
        false,        // delete when unused
        false,        // exclusive
        false,        // no-wait
        nil,          // arguments
    )
    failOnError(err, "Failed to declare a queue")

    if (*veryVerbose) {
        log.Printf("There are %d messages in queue %s", toQueue.Messages, toQueue.Name)
    }

    msgs, err := ch.Consume(
        fromQueue.Name, // queue
        "",             // consumer
        false,          // auto-ack (it's very important this stays false)
        false,          // exclusive
        false,          // no-local
        false,          // no-wait
        nil,            // args
    )
    failOnError(err, "Failed to register the scoop consumer")

    log.Printf("Running scoop consumer... (press Ctl-C to cancel)")

    i := 1

    for d := range msgs {
        if (i > *messageCount) {
            if (*extremelyVerbose) {
                log.Printf("Complete")
            }
            break
        }

        err = ch.Publish(
            *exchange,    // exchange
            toQueue.Name, // routing key
            false,        // mandatory
            false,        // immediate
            amqp.Publishing{
                ContentType: "text/plain",
                Body:        []byte(d.Body),
            })

        failOnError(err, "Failed to deliver message")

        if (*extremelyVerbose) {
            log.Printf("Successfully delivered message (%d/%d)", i, *messageCount)
        }

        d.Ack(true)
        i++
    }
}

func failOnError(err error, msg string) {
    if err != nil {
        log.Fatalf("%s: %s", msg, err)
    }
}
