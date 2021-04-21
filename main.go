package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var settings struct {
	ServerHost                 string
	ServerPort                 uint16
	ServerPassword             string
	SourceAddress              string
	AppID                      uint32
	ServerTimeoutSeconds       int
	RecTalkgroupID             uint32
	CallHangTimeSeconds        int
	CallExecCommand1           string
	CallExecCommand1ShowStderr bool
	CallExecCommand2           string
	CallExecCommand2ShowStderr bool
	CallExecCommand3           string
	CallExecCommand3ShowStderr bool
	OutputDir                  string
	OutputFileExtension        string
	CreateDailyAggregateFile   bool
}

var loggedIn bool

type udpPacket struct {
	data []byte
	len  int
}

// receivePackets sends all received packets on the given connection to the given channel.
func receivePackets(conn net.Conn, recvUDP chan udpPacket) {
	for {
		buffer := make([]byte, 128)
		readBytes, err := conn.Read(buffer)
		if err != nil {
			log.Fatal(err)
		}
		recvUDP <- udpPacket{data: buffer, len: readBytes}
	}
}

// handlePacket returns true if given packet was valid.
func handlePacket(conn net.Conn, p *udpPacket) bool {
	var rd rewindData
	rb := bytes.NewReader(p.data)
	binary.Read(rb, binary.LittleEndian, &rd)
	payload := make([]byte, rd.PayloadLength)
	pl, err := rb.Read(payload)
	if err != nil || pl != int(rd.PayloadLength) {
		log.Println("invalid payload length, dropping packet")
		return false
	}

	switch rd.PacketType {
	case rewindPacketTypeKeepAlive:
		//log.Println("got keepalive response")

		if !loggedIn {
			// Requesting super headers.
			sendConfiguration(conn, rewindOptionSuperHeader)
		}
	case rewindPacketTypeConfiguration:
		log.Println("got configuration ack")
		if !loggedIn {
			// Subscribing to the requested TG.
			sendSubscription(conn, settings.RecTalkgroupID, rewindSessionTypeGroupVoice)
		}
	case rewindPacketTypeSubscription:
		log.Println("got subscription ack")
		if !loggedIn {
			log.Println("logged in")
			loggedIn = true
		}
	case rewindPacketTypeReport:
		log.Println("server report: ", pl)
	case rewindPacketTypeChallenge:
		log.Println("got challenge")
		loggedIn = false
		sendChallengeResponse(conn, sha256.Sum256(append(payload, []byte(settings.ServerPassword)...)))
	case rewindPacketTypeSuperHeader:
		//log.Println("got super header")
		var sh rewindSuperHeader
		rb = bytes.NewReader(payload)
		binary.Read(rb, binary.LittleEndian, &sh)
		if callData.ongoing && sh != callData.lastSuperHeader {
			handleCallEnd()
		}
		if !callData.ongoing {
			handleCallStart(sh)
		}
	case rewindPacketTypeDMRTerminatorWithLC:
		//log.Println("got dmr terminator with lc")
		handleCallEnd()
	case rewindPacketTypeFailureCode:
		log.Println("got failure code: ", pl)
	case rewindPacketTypeDMRAudioFrame:
		//log.Println("got dmr audio frame")
		handleDMRAudioFrame(payload)
	case rewindPacketTypeClose:
		log.Fatal("got close request")
	default:
		return false
	}
	return true
}

func main() {
	configFileName := "config.json"

	flag.StringVar(&configFileName, "c", configFileName, "config file to use, default: config.json")
	flag.Parse()

	logFile, err := os.OpenFile("callrec.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Println("warning: can't open callrec.log for writing: ", err)
	} else {
		defer logFile.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	cf, err := os.Open(configFileName)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.NewDecoder(cf).Decode(&settings); err != nil {
		log.Fatal("error parsing config file:", err.Error())
	}

	serverHostPort := fmt.Sprintf("%s:%d", settings.ServerHost, settings.ServerPort)
	log.Println("using server and port", serverHostPort)
	raddr, err := net.ResolveUDPAddr("udp", serverHostPort)
	if err != nil {
		log.Fatal(err)
	}
	var laddr *net.UDPAddr
	if (settings.SourceAddress != "") {
		log.Println("Using custom listening address " + settings.SourceAddress)
		laddr, err = net.ResolveUDPAddr("udp", settings.SourceAddress)
		if err != nil {
			log.Fatal(err)
		}
	}

	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	recvUDP := make(chan udpPacket)
	go receivePackets(conn, recvUDP)

	log.Println("starting listening loop")

	var timeLastSentKeepalive time.Time
	var timeLastValidPacket time.Time

	for {
		timeDiff := time.Since(timeLastSentKeepalive)
		if timeDiff.Seconds() >= 5 {
			sendKeepalive(conn)
			timeLastSentKeepalive = time.Now()
		}

		select {
		case p := <-recvUDP:
			if p.len >= len(rewindProtocolSign) && bytes.Compare(p.data[:len(rewindProtocolSign)], []byte(rewindProtocolSign)) == 0 {
				if handlePacket(conn, &p) {
					timeLastValidPacket = time.Now()
				}
			}
		case <-time.After(time.Second * 5):
		}

		timeDiff = time.Since(timeLastValidPacket)
		if timeDiff.Seconds() >= float64(settings.ServerTimeoutSeconds) {
			log.Fatal("timeout, disconnected")
		}

		if callData.ongoing {
			timeDiff = time.Since(callData.lastFrameReceived)
			if timeDiff.Seconds() >= float64(settings.CallHangTimeSeconds) {
				log.Println("call timeout")
				handleCallEnd()
			}
		}
	}
}
