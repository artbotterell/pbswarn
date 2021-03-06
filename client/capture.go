/**************************************************************
 *
 *  Copyright (c) 2019 Public Broadcasting Service
 *  Contact: <warn@pbs.org>
 *  All Rights Reserved.
 *
 *  Version 1.4 4/21/2019
 *
 *************************************************************/

package main

import (
	"bytes"
	"database/sql"
	"hash"
	"log"
	"logging"
	"net"
	"strconv"
	"strings"

	"../cap"
	"../warndb"

	config "github.com/Tkanos/gonfig"
	"golang.org/x/net/ipv4"
)

const postURL = "https://94e38d27ol.execute-api.us-west-2.amazonaws.com/dev"

var (
	inMessage bool
	message   []byte
	breaker   []byte
	lastHash  string
	replacer  strings.Replacer
	h         hash.Hash
	db        *sql.DB
	err       error
	result    sql.Result
	rows      *sql.Rows
	dispPoly  string
	channel   chan []byte
	cfg       Configuration
)

// Configuration data structure for application config
type Configuration struct {
	Station   string
	Multicast string
}

type mapItem struct {
	ID           string
	Sent         string
	Status       string
	MsgType      string
	Cmam         string
	Headline     string
	Source       string
	Levels       string
	ResponseType string
	Description  string
	Instruction  string
	Expires      string
	AreaDesc     string
	Geocodes     []string
	Polygons     string
}

// Run is a goroutine to monitor UDP packets from the WARN receiver and
// re-assemble
func main() {
	// Get configuration
	cfg = Configuration{}
	if err := config.GetConf("./warnRx.conf", &cfg); err != nil {
		logging.Log("warnRx.main GetConf", err.Error())
	}
	inMessage = false
	breaker = []byte{0x47, 0x09, 0x11} // Start of MPEG Packet break?
	if err != nil {
		log.Fatal("Create breaker", err)
	}
	breaker = []byte{0x47, 0x09, 0x11} // Start of MPEG Packet break?
	// set up the UDP monitor
	eth0, err := net.InterfaceByName("eth0")
	if err != nil {
		log.Fatal("InterfaceByName", err)
	}
	adr := strings.Split(cfg.Multicast, ":")
	ad := strings.Split(adr[0], ".")
	a, _ := strconv.Atoi(ad[0])
	b, _ := strconv.Atoi(ad[1])
	c, _ := strconv.Atoi(ad[2])
	d, _ := strconv.Atoi(ad[3])

	group := net.IPv4(byte(a), byte(b), byte(c), byte(d))
	conn, err := net.ListenPacket("udp4", cfg.Multicast)
	if err != nil {
		log.Fatal("ListenPacket", err)
	}
	defer conn.Close()
	p := ipv4.NewPacketConn(conn)
	if err := p.JoinGroup(eth0, &net.UDPAddr{IP: group}); err != nil {
		log.Fatal("JoinGroup", err)
	}
	bs := make([]byte, 1500)
	log.Println("Starting multicast monitoring from WARN receiver on " + cfg.Multicast + "\n")
	for {
		n, _, _, _ := p.ReadFrom(bs)
		packetHandler(bs, n)
	}
}

// process each multicast packet, pass along those for assembly
func packetHandler(b []byte, n int) {
	msg := b[24:n]
	msg = removeAll(msg, breaker) // take out MPEG packet breaks
	assemble(msg)
}

func assemble(msg []byte) {

	// if packet contains "<?xml ", start a new message
	st := bytes.Index(msg, []byte("<?xml "))
	if st != -1 {
		if !inMessage {
			inMessage = true
			msg = msg[st:] // trim off leading garbage
		}
		message = make([]byte, 0) // init a new message
	}

	// if packet contains "</ale" it's the end of the message
	en := bytes.Index(msg, []byte("</ale"))
	if inMessage && en != -1 {
		msg = msg[:en]                           // trim off trailing garbage
		msg = append(msg, []byte("</alert>")...) // repair the closing tag
		message = append(message, msg...)
		inMessage = false
		alert := cap.ParseCAP([]byte(message))
		warndb.ToDB(alert, string(message))
		warndb.Uptime()
		return
	}

	// likewise if packet contains  "</CMAC_Alert_Attr" it's the end of the message
	en = bytes.Index(msg, []byte("</CMAC_Alert_Attr"))
	if inMessage && en != -1 {
		msg = msg[:en]                                           // trim off trailing garbage
		msg = append(msg, []byte("</CMAC_Alert_Attributes>")...) // repair the closing tag
		message = append(message, msg...)
		inMessage = false
		log.Println("CMAM")
		warndb.Uptime()
		return
	}

	// otherwise, if we're inside a message, append it
	if inMessage {
		message = append(message, msg...)
	}
}

// remove all instances of a byte slice from within another byte slice
func removeAll(source []byte, remove []byte) []byte {
	for bytes.Index(source, remove) > -1 {
		pnt := bytes.Index(source, remove)
		source = append(source[:pnt], source[pnt+12:]...)
	}
	return source
}
