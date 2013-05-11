package main

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"koding/kontrol/helper"
	"koding/kontrol/kontroldaemon/workerconfig"
	"koding/tools/process"
	"log"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

func init() {
	log.SetPrefix("kontrold-client ")
}

func main() {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     "",
		done:    make(chan error),
	}
	c.conn = helper.CreateAmqpConnection()
	c.channel = helper.CreateChannel(c.conn)

	err := c.channel.ExchangeDeclare("clientExchange", "fanout", true, false, false, false, nil)
	if err != nil {
		log.Fatal("info exchange.declare: %s", err)
	}

	if _, err := c.channel.QueueDeclare("", false, true, false, false, nil); err != nil {
		log.Fatal("clientProducer queue.declare: %s", err)
	}

	if err := c.channel.QueueBind("", "", "clientExchange", false, nil); err != nil {
		log.Fatal("clientProducer queue.bind: %s", err)
	}

	stream, err := c.channel.Consume("", "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("clientProducer basic.consume: %s", err)
	}

	go handle(stream, c)

	log.Println("starting to listen for requests...")

	select {}

}

func handle(deliveries <-chan amqp.Delivery, consumer *Consumer) {
	for d := range deliveries {
		log.Printf("handle got %dB message data: [%v] %s %s",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
			d.AppId)

		var req workerconfig.ClientRequest
		err := json.Unmarshal(d.Body, &req)
		if err != nil {
			log.Print("bad json incoming msg: ", err)
		}

		matchAction(req.Action, req.Cmd, req.Hostname, req.Pid)

	}

	log.Printf("handle deliveries channel closed")
	consumer.done <- nil
}

func matchAction(action, cmd, hostname string, pid int) {
	funcs := map[string]func(cmd, hostname string, pid int) error{
		"start": start,
		"check": check,
		"kill":  kill,
		"stop":  stop,
	}

	if hostname != "" && hostname != helper.CustomHostname() {
		log.Println("command is for a different machine")
		return
	}

	if pid == 0 && action != "start" {
		log.Println("please provide pid number for '%s'", action)
	}

	err := funcs[action](cmd, hostname, pid)
	if err != nil {
		log.Println("call function err", err)
	}

}

func start(cmd, hostname string, pid int) error {
	log.Printf("trying to start command '%s'", cmd)

	_, err := process.RunCmd(cmd)
	if err != nil {
		return err
	}

	log.Printf("cmd '%s' started", cmd)
	return nil
}

func check(cmd, hostname string, pid int) error {
	err := process.CheckPid(pid)
	if err != nil {
		return err
	}

	log.Printf("local process with %s pid is alive", pid)
	return nil
}

func kill(cmd, hostname string, pid int) error {
	err := process.KillCmd(pid)
	if err != nil {
		return err
	}

	log.Printf("local process with %s pid is killed", pid)
	return nil
}

func stop(cmd, hostname string, pid int) error {
	err := process.StopPid(pid)
	if err != nil {
		return err
	}

	log.Printf("local process with %s pid is get SIGSTOP", pid)
	return nil
}
