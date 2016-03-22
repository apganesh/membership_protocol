package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

var (
	mu       sync.Mutex
	logmu    sync.Mutex
	cm       *Member
	udpConn  *net.UDPConn
	observer *Observer
	Eventlog []string
	quitChan chan bool
)

type config struct {
	Tgossip   time.Duration `json:"gossip_frequency"`
	Tfail     time.Duration `json:"fail_timeout"`
	Tcleanup  time.Duration `json:"cleanup_timeout"`
	Heartbeat int
}

var defaultConfig = config{
	Tgossip:   2000,
	Tfail:     5000,
	Tcleanup:  20000,
	Heartbeat: 2,
}

type IncomingMessage struct {
	rAddr net.UDPAddr
	buf   []byte
}

func main() {

	addr := flag.String("addr", "", "Server address (by default picks up local ip address)")
	port := flag.Int("port", 0, "Server port(by default picks up 8900)")
	masternode := flag.Bool("masternode", false, "Is this deamon for master node")
	masteraddr := flag.String("masteraddr", "", "Master node IP address")
	configfile := flag.String("config", "", "Config file path")

	flag.Parse()

	// Seed the random
	rand.Seed(time.Now().UnixNano())

	if *addr == "" {
		*addr = getLocalIPAddress()
	}
	if *port == 0 {
		*port = 8900
	}
	if *masteraddr == "" {
		*masteraddr = *addr + ":" + "8900"
	}

	if *configfile != "" {
		loadConfigFile(*configfile, &defaultConfig)
	}

	ipAddr := *addr + ":" + strconv.Itoa(*port)

	udpAddr := &net.UDPAddr{Port: *port, IP: net.ParseIP(*addr)}

	fmt.Println("Starting UDP server at: ", udpAddr.IP.String(), udpAddr.Port)

	var err error
	udpConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}
	defer udpConn.Close()

	if *masternode {
		cm = NewMember(ipAddr, MASTER, RUNNING, 1)
	} else {
		cm = NewMember(ipAddr, NODE, RUNNING, 1)
		udpMasterAddr := getUDPAddr(*masteraddr)
		cm.sendUDPMemberList(udpMasterAddr)
	}

	udpMsgChan := make(chan IncomingMessage)
	quitChan = make(chan bool)

	// Handle SIGINT and SIGTERM.
	signalChannel := make(chan os.Signal, 1)
	go signalHandler(signalChannel)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Run the Heartbeat daemon
	go runHeartBeatDaemon(udpMsgChan)

	// Handle incoming UDP data from the other members
	go handleUDPMessages(udpMsgChan)

	// Setup the HTTP server
	go startHTTPServer(ipAddr)

	// Setup the metrics event publisher
	go func() {

		tt := time.Tick(1 * time.Second)
		for {
			select {
			case <-tt:
				publishMetrics()
			}
		}
	}()

	<-quitChan
}

func startHTTPServer(ipAddr string) {
	// HTTP api for the server
	r := mux.NewRouter()
	observer = NewObserver()

	r.HandleFunc("/terminate", TerminateHandler)

	// Route these commands only for the master
	// if cm.Type == MASTER {
	// 	r.HandleFunc("/memberlist/{id}", MemberListHandler)
	// 	r.HandleFunc("/eventlist/{id}", EventListHandler)
	// 	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./html/")))
	// }
	r.HandleFunc("/memberlist/{id}", MemberListHandler)
	r.HandleFunc("/eventlist/{id}", EventListHandler)
	r.Handle("/events", observer)
	//r.Handle("/cpu", cpuobserver)
	//r.HandleFunc("/events", ob)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./html/")))

	http.Handle("/", r)

	fmt.Println("Starting the HTTP server")
	err := http.ListenAndServe(ipAddr, r)
	if err != nil {
		panic(err)
	}
}
