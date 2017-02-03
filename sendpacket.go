package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

var txSeqNum uint32

func sendKeepalive(conn net.Conn) {
	//log.Println("sending keepalive")

	var rd rewindData
	copy(rd.Sign[:], []byte(rewindProtocolSign))
	rd.PacketType = rewindPacketTypeKeepAlive
	rd.PayloadLength = uint16(rewindVersionDataLength)
	rd.SeqNum = txSeqNum
	txSeqNum++

	var rvd rewindVersionData
	rvd.RemoteID = settings.AppID
	rvd.RewindService = rewindServiceSimpleApplication
	copy(rvd.Description[:], []byte(rewindVersionDescription))

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, &rd)
	binary.Write(&buf, binary.LittleEndian, &rvd)
	writtenBytes, err := conn.Write(buf.Bytes())
	if writtenBytes != rewindDataLength+int(rd.PayloadLength) || err != nil {
		log.Println("warning: can't send udp packet", err)
	}
}

func sendChallengeResponse(conn net.Conn, resp [32]byte) {
	log.Println("sending challenge response")

	var rd rewindData
	copy(rd.Sign[:], []byte(rewindProtocolSign))
	rd.PacketType = rewindPacketTypeAuthentication
	rd.PayloadLength = 32
	rd.SeqNum = txSeqNum
	txSeqNum++

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, &rd)
	buf.Write(resp[:])
	writtenBytes, err := conn.Write(buf.Bytes())
	if writtenBytes != rewindDataLength+int(rd.PayloadLength) || err != nil {
		log.Println("warning: can't send udp packet", err)
	}
}

func sendConfiguration(conn net.Conn, conf uint32) {
	log.Println("sending configuration")

	var rd rewindData
	copy(rd.Sign[:], []byte(rewindProtocolSign))
	rd.PacketType = rewindPacketTypeConfiguration
	rd.PayloadLength = 4
	rd.SeqNum = txSeqNum
	txSeqNum++

	var rcd rewindConfigurationData
	rcd.Options = conf

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, &rd)
	binary.Write(&buf, binary.LittleEndian, &rcd)
	writtenBytes, err := conn.Write(buf.Bytes())
	if writtenBytes != rewindDataLength+int(rd.PayloadLength) || err != nil {
		log.Println("warning: can't send udp packet", err)
	}
}

func sendSubscription(conn net.Conn, dstID uint32, sessionType uint32) {
	log.Println("sending subscription")

	var rd rewindData
	copy(rd.Sign[:], []byte(rewindProtocolSign))
	rd.PacketType = rewindPacketTypeSubscription
	rd.PayloadLength = 8
	rd.SeqNum = txSeqNum
	txSeqNum++

	var rsd rewindSubscriptionData
	rsd.DstID = dstID
	rsd.SessionType = sessionType

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, &rd)
	binary.Write(&buf, binary.LittleEndian, &rsd)
	writtenBytes, err := conn.Write(buf.Bytes())
	if writtenBytes != rewindDataLength+int(rd.PayloadLength) || err != nil {
		log.Println("warning: can't send udp packet", err)
	}
}
