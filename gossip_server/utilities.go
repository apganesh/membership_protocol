package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"strconv"
)

func getUDPAddr(addr string) *net.UDPAddr {

	ip, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)
	udpAddr := &net.UDPAddr{IP: net.ParseIP(ip), Port: port}
	return udpAddr
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
	return "127.0.0.1"
}

func loadConfigFile(path string, c *config) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Config File Missing. ", err)
	}

	//var config Configuration
	err = json.Unmarshal(file, c)
	if err != nil {
		log.Fatal("Config Parse Error: ", err)
	}
}
