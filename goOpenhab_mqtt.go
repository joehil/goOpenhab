package main

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func publishMqtt(mess chan Mqttparms) {
	broker := genVar.Mqttbroker
	var topic string
	var message string

	var clientId = "go_mqtt_client"
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientId)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		traceLog(fmt.Sprintln(token.Error()))
	}

	// Subscribe to a topic
	subtopic := "goopenhab/intopic"
	qos := 1
	if token := client.Subscribe(subtopic, byte(qos), nil); token.Wait() && token.Error() != nil {
		traceLog(fmt.Sprintln(token.Error()))
	}

	for {
		// Publish a message
		inmsg := <-mess
		topic = inmsg.Topic
		message = inmsg.Message
		token := client.Publish(topic, byte(qos), false, message)
		token.Wait()
		debugLog(5,fmt.Sprintf("Message published to topic %s: %s", topic, message))
		time.Sleep(1 * time.Second)
	}
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	traceLog(fmt.Sprintf("mqtt message received: %s from topic: %s", msg.Payload(), msg.Topic()))
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	traceLog(fmt.Sprintln("mqtt connected"))
}

// Modified connectLostHandler with reconnect logic
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	traceLog(fmt.Sprintf("mqtt connection lost: %v", err))
	traceLog(fmt.Sprint("mqtt attempting to reconnect..."))
	for {
		time.Sleep(5 * time.Second) // Wait for 5 seconds before trying to reconnect
		if token := client.Connect(); token.Wait() && token.Error() == nil {
			traceLog(fmt.Sprintln("mqtt reconnected"))
			break // Exit the loop once reconnected
		} else {
			traceLog(fmt.Sprintf("mqtt reconnect attempt failed: %v", token.Error()))
			// You may choose to implement additional logic to limit the number of retries or to handle failures differently
		}
	}
}
