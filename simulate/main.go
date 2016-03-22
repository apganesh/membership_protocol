package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var (
	mu          sync.Mutex
	processed   []string
	unprocessed []string
	Tup         time.Duration
	Tdown       time.Duration
	mNodeIP     string
)

func init() {
	Tup = 8 * time.Second
	Tdown = 15 * time.Second
}

func getLocalIPAddress() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func signalHandler(ch chan os.Signal) {
	sig := <-ch
	switch sig {
	case os.Interrupt:
		log.Println("Got an interrupt")
		bringDownMachines()
		os.Exit(1)
	case syscall.SIGTERM:
		os.Exit(1)
	}
}
func startMaster(port string) {
	lAddr := getLocalIPAddress()
	mNodeIP = lAddr + ":" + port

	cmd := exec.Command("bin/gossip_server", "-masternode", "-addr", lAddr, "-port", port, "-config", "cfg/gossip_config.json")
	cmd.Stdout = os.Stdout
	err := cmd.Start()

	log.Println("Starting Master Node ", mNodeIP)
	if err != nil {
		log.Println("Had an error in startMaster", err)
		log.Fatal(err)
	}
	cmd.Wait()
}

func initializeMachines() {
	lAddr := getLocalIPAddress()
	for i := 8911; i < 8921; i++ {
		ip := lAddr + ":" + strconv.Itoa(i)
		unprocessed = append(unprocessed, ip)
	}

}

func startServer(ipaddr string) {
	addr, port, _ := net.SplitHostPort(ipaddr)
	log.Println("Starting  Node: ", addr, port)
	cmd := exec.Command("bin/gossip_server", "-masteraddr", mNodeIP, "-addr", addr, "-port", port, "-config", "cfg/gossip_config.json")
	//cmd.Stdout = os.Stdout
	err := cmd.Start()

	if err != nil {
		log.Println("Error occured for startServer: ", ipaddr, err)
		log.Fatal(err)
	}

	cmd.Wait()
}

func stopServer(ipaddr string) {

	cmd := exec.Command("curl", "-i", "http://"+ipaddr+"/terminate")
	log.Println("Stopping Node: ", ipaddr)
	err := cmd.Start()

	if err != nil {
		log.Println("Error occured for stopServer: ", ipaddr)
		log.Fatal(err)
	}
	cmd.Wait()
	//time.Sleep(3 * time.Second)
}

func bringUpMachines() {
	//fmt.Println("Inside bring up machines")
	mu.Lock()
	defer mu.Unlock()

	if len(unprocessed) == 0 {
		return
	}

	nr := rand.Intn(10)

	if nr > 4 {
		return
	}

	for i := 0; i < 4; i++ {

		r := rand.Intn(len(unprocessed))
		addr := unprocessed[r]

		go startServer(addr)
		processed = append(processed, addr)
		unprocessed = append(unprocessed[:r], unprocessed[r+1:]...)

		if len(unprocessed) == 0 {
			break
		}

	}

}

func bringDownMachines() {
	//fmt.Println("Inside bring down machines")
	mu.Lock()
	defer mu.Unlock()
	if len(processed) < 5 {
		return
	}

	nr := rand.Intn(10)

	if nr > 4 {
		return
	}

	for i := 0; i < 3; i++ {
		r := rand.Intn(len(processed))
		addr := processed[r]
		go stopServer(addr)
		unprocessed = append(unprocessed, addr)
		processed = append(processed[:r], processed[r+1:]...)
		if len(processed) == 0 {
			break
		}

	}

}

func terminateMachines() {
	mu.Lock()
	defer mu.Unlock()
	fmt.Println("Terminating all active nodes")
	unprocessed = nil
	for _, ipaddr := range processed {
		stopServer(ipaddr)
	}
	processed = nil
}

func bringDownMachines_Timer() {
	hb := time.Tick(Tdown)
	for {
		select {
		case <-hb:
			bringDownMachines()
		}
	}
}

func bringUpMachines_Timer() {
	hb := time.Tick(Tup)
	for {
		select {
		case <-hb:
			bringUpMachines()
		}
	}
}

func terminateSimulator_Timer() {

	shutdown_timer := time.NewTimer(5 * time.Minute)
	masternode_timer := time.NewTimer(6 * time.Minute)
	for {
		select {
		case <-shutdown_timer.C:
			terminateMachines()
		case <-masternode_timer.C:
			fmt.Println("Terminating the main Node")
			stopServer(mNodeIP) // Stop the main node after 1 minute to let the other nodes cleanup
			os.Exit(1)          // Need to do a graceful cleanup.
		}

	}

}

func main() {
	// Initialize the randowm seed
	rand.Seed(time.Now().Unix())

	// Handle SIGINT and SIGTERM.
	signalChannel := make(chan os.Signal, 1)
	go signalHandler(signalChannel)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	quitCh := make(chan bool)

	initializeMachines()

	var port = os.Getenv("PORT")
	if port == "" {
		port = "7700"
	}

	go startMaster(port)
	time.Sleep(2 * time.Second)
	fmt.Println("Starting simulateFailures")
	//go simulateFailures()
	go terminateSimulator_Timer()

	go bringUpMachines_Timer()
	// time.Sleep(2 * time.Second)
	go bringDownMachines_Timer()

	fmt.Println("Waiting for quit signal")
	<-quitCh
}
