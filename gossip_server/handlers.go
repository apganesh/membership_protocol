package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func ErrorHandler(rw http.ResponseWriter, req *http.Request) {
	body, _ := ioutil.ReadFile("errors/index.html")
	fmt.Fprint(rw, string(body))
}

func TerminateHandler(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Got terminate signal: " + cm.IPAddress)
	// Close all the channels on the observer
	quitChan <- true
}

func MemberListHandler(rw http.ResponseWriter, req *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	vars := mux.Vars(req)
	id, _ := strconv.Atoi(vars["id"])

	var bcMembers Members
	if cm.eventid > id {
		for _, mbr := range cm.memberList {
			bcMembers = append(bcMembers, mbr)
		}
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewEncoder(rw).Encode(bcMembers)

}

type JsonLogs struct {
	Id   int
	Logs []string
}

func EventListHandler(rw http.ResponseWriter, req *http.Request) {

	mu.Lock()
	defer mu.Unlock()

	vars := mux.Vars(req)
	id, _ := strconv.Atoi(vars["id"])

	var logs JsonLogs

	if cm.eventid > id {
		logs.Id = cm.eventid
		logs.Logs = Eventlog[id:]
	} else {
		logs.Id = id
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(rw).Encode(logs)
}

func handleUDPMessages(udpMsgChan chan IncomingMessage) {
	fmt.Println("Waiting for UDP connection")
	for {
		buf := make([]byte, 4096)
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
		}
		udpMsgChan <- IncomingMessage{*addr, buf[:n]}
	}

}

func metricsDaemon() {

	tt := time.Tick(1 * time.Second)
	for {
		select {
		case <-tt:
			publishMetrics()
		}
	}
}

func signalHandler(ch chan os.Signal) {
	sig := <-ch
	switch sig {
	case os.Interrupt:
		os.Exit(1)
	case syscall.SIGTERM:
		quitChan <- true
	}
}
