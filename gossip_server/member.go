package main

import (
	"encoding/json"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	TimeLayout = "01/02/2006 15:04:05 MST"
)

type TimeFormat struct {
	time.Time
}

type Member struct {
	IPAddress  string
	Type       int `json:"-"`
	Status     int
	Heartbeat  int
	Timestamp  TimeFormat
	eventid    int                `json:"-"`
	memberList map[string]*Member `json:"-"`
}

type Members []*Member

type Event struct {
	IPAddress string
	Timestamp TimeFormat
	Status    int
}

// For node type
const (
	MASTER = 1
	NODE   = 2
)

// For status
const (
	RUNNING  = 1
	FAILING  = 2
	FAILED   = 3
	REJOINED = 4
)

// http://blog.charmes.net/2015/08/json-date-management-in-golang_7.html
func (t *TimeFormat) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Time.Format(TimeLayout) + `"`), nil
}

func (t *TimeFormat) UnmarshalJSON(buf []byte) error {
	tt, err := time.Parse(TimeLayout, strings.Trim(string(buf), `"`))
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}

func NewMember(address string, nodetype int, status int, heartbeat int) *Member {
	newm := &Member{address, nodetype, status, heartbeat, TimeFormat{time.Now().UTC().Local()}, 0, nil}
	newm.memberList = make(map[string]*Member)
	return newm
}

func (m *Member) addMember(address string, nodetype int, status int, heartbeat int) {
	newm := NewMember(address, nodetype, status, heartbeat)
	m.memberList[address] = newm
}

func addLog(addr string, ctime TimeFormat, status int) {
	logmu.Lock()
	defer logmu.Unlock()
	var evts []Event
	//ltime := TimeFormat{time.Now().UTC()}

	event := Event{addr, ctime, status}
	evts = append(evts, event)
	b, _ := json.Marshal(evts)
	observer.publishMessage(Message{"status", b})
}

func (m *Member) updateMemberList(buf []byte) {
	mu.Lock()
	defer mu.Unlock()

	var mbrs Members
	err := json.Unmarshal(buf, &mbrs)
	if err != nil {
		panic(err)
	}

	ctime := TimeFormat{time.Now().UTC().Local()}

	for _, mbr := range mbrs {
		if m.IPAddress == mbr.IPAddress {
			continue
		}
		mem, ok := m.memberList[mbr.IPAddress]
		if !ok {
			//nm := NewMember(mbr.IPAddress, mbr.Type, mbr.Heartbeat)
			m.addMember(mbr.IPAddress, mbr.Type, mbr.Status, mbr.Heartbeat)
			addLog(mbr.IPAddress, ctime, RUNNING)
			//nm.Eventtime = time.Now().Format(time.RFC1123)
		} else {
			// This is for re-joining
			if mbr.Heartbeat > mem.Heartbeat && mem.Status == RUNNING {
				mem.Heartbeat = mbr.Heartbeat
				mem.Timestamp = ctime
				mem.Status = RUNNING
			} else if mbr.Heartbeat < mem.Heartbeat && mem.Status != RUNNING {
				mem.Heartbeat = mbr.Heartbeat
				mem.Status = RUNNING
				mem.Timestamp = ctime
				//addLog("RE-JOINED " + mbr.IPAddress)
				addLog(mbr.IPAddress, ctime, REJOINED)
				//mem.Eventtime = time.Now().Format(time.RFC1123)
			} else if mem.Status == RUNNING && mem.Heartbeat > mbr.Heartbeat {
				mem.Timestamp = ctime
			}

		}
	}

}

func (m *Member) purgeExpiredMembers() {
	ctime := TimeFormat{time.Now().UTC().Local()}
	m.Timestamp = ctime

	for _, mbr := range m.memberList {
		var diff time.Duration = (ctime.Time.Sub(mbr.Timestamp.Time))
		if diff > (defaultConfig.Tcleanup*time.Millisecond) && mbr.Status == FAILING {
			addLog(mbr.IPAddress, ctime, FAILED)
			mbr.Status = FAILED
		} else if mbr.Status == RUNNING && diff > (defaultConfig.Tfail*time.Millisecond) {
			addLog(mbr.IPAddress, ctime, FAILING)
			mbr.Status = FAILING
		}
	}
}

func yatesShuffle(addrs []*net.UDPAddr) []*net.UDPAddr {
	for i := len(addrs); i > len(addrs)/2; i-- {
		r := rand.Intn(i)
		addrs[i], addrs[r] = addrs[r], addrs[i]
	}
	return addrs
}

func (m *Member) marshalMemberList() []byte {
	var members Members
	members = append(members, m)
	for _, mbr := range m.memberList {
		members = append(members, mbr)
	}
	myjson, _ := json.Marshal(members)

	return myjson
}

func (m *Member) sendUDPMemberList(udpAddr *net.UDPAddr) {
	mlist := m.marshalMemberList()
	udpConn.WriteToUDP([]byte(mlist), udpAddr)
}

func (m *Member) broadcastMemberList() {
	mu.Lock()
	defer mu.Unlock()

	if len(m.memberList) == 0 {
		return
	}

	m.purgeExpiredMembers()

	myjson := m.marshalMemberList()

	var addrs []*net.UDPAddr

	// : Pick random nodes to broadcast
	for _, mbr := range cm.memberList {
		addr, p, _ := net.SplitHostPort(mbr.IPAddress)
		port, _ := strconv.Atoi(p)
		addrs = append(addrs, &net.UDPAddr{Port: port, IP: net.ParseIP(addr)})
	}

	for _, rAddr := range addrs {
		udpConn.WriteToUDP([]byte(myjson), rAddr)
	}
}

func runHeartBeatDaemon(inMsg chan IncomingMessage) {
	hb := time.Tick(defaultConfig.Tgossip * time.Millisecond)

	for {
		select {
		case x := <-inMsg:
			cm.updateMemberList(x.buf)
		case <-hb:
			cm.Heartbeat += defaultConfig.Heartbeat
			cm.broadcastMemberList()
		}
	}

}
