package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

type Message struct {
	eventType string
	content   []byte
}

type Observer struct {
	subscribers map[chan Message]bool
}

func NewObserver() *Observer {
	s := make(map[chan Message]bool)
	return &Observer{s}
}

func (ob *Observer) addSubscriber() chan Message {
	ch := make(chan Message)
	ob.subscribers[ch] = true
	return ch
}

func (ob *Observer) delSubscriber(ch chan Message) {
	delete(ob.subscribers, ch)
}

func (ob *Observer) Listen() {

}

func (ob *Observer) publishMessage(msg Message) {

	for ch := range ob.subscribers {
		ch <- msg
	}
}

func publishMetrics() {

	type NodeMetrics struct {
		IPAddress string
		CPULoad   int
		MemUsage  int
	}

	cpuload := rand.Intn(100)
	memusage := rand.Intn(100)

	var metrics []*NodeMetrics
	metrics = append(metrics, &NodeMetrics{cm.IPAddress, cpuload, memusage})
	metricsjson, _ := json.Marshal(metrics)
	observer.publishMessage(Message{"metrics", metricsjson})
}

func (ob *Observer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	// Get the flusher working
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming events not supported", http.StatusServiceUnavailable)
		return
	}

	//Setup the header
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// New connections will subscribe themselves and defer the unsubscribe
	ch := ob.addSubscriber()
	quitCh := make(chan bool)

	// Setup the Notifier
	notify := rw.(http.CloseNotifier).CloseNotify()

	go func() {
		<-notify
		ob.delSubscriber(ch)
	}()

	// Wait for the events
	go func() {
		for {
			msg, _ := <-ch
			fmt.Fprintf(rw, "event: %s\n", msg.eventType)
			fmt.Fprintf(rw, "data: %s\n\n", msg.content)

			flusher.Flush()
		}
	}()
	ch <- Message{"status", cm.marshalMemberList()}

	<-quitCh
}
