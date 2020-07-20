package main

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type RTTStruct struct {
	totalRTT int64
	samples  int64
	avgRTT   float64
}

func RTTHandler(s network.Stream, host host.Host) {
	fmt.Println("Got a stream!")
	defer s.Close()
	buf := bufio.NewReader(s)
	data, err := buf.ReadString('\n')
	ackStr := strings.Split(data, "\n")[0]
	timestampStr := strings.Split(ackStr, ".")[1]

	if err != nil {
		fmt.Println(err)
		s.Reset()
	}
	fmt.Println("Received message: ", data)
	_, err = s.Write([]byte(fmt.Sprintf("ACK.%s.%s\n", timestampStr, host.ID().String())))
	if err != nil {
		fmt.Println(err)
		s.Reset()
	}
}

func sendPingLoop(host host.Host) {
	fmt.Println("Starting ping loop")
	rttTable := map[string]*RTTStruct{}

	for {
		// For each node that connects.
		for _, peer := range host.Network().Peers() {
			sendPing(host, peer, rttTable)
		}
		time.Sleep(5 * time.Second)
	}
}

func sendPing(host host.Host, peer peer.ID, rttTable map[string]*RTTStruct) {
	startTime := time.Now().UnixNano()
	// Opening stream
	s, err := host.NewStream(context.Background(), peer, "/rtt/0.0.1")
	if err != nil {
		fmt.Println("Couldn't create the stream")
		return
	}
	// Send ping over opened stream
	_, err = s.Write([]byte(fmt.Sprintf("ping.%s\n", strconv.FormatInt(startTime, 10))))
	if err != nil {
		fmt.Println("Error sending ping", err)
	}
	// Read ACK
	buf := bufio.NewReader(s)
	data, err := buf.ReadString('\n')
	// Collect data from ACK
	ackStr := strings.Split(data, "\n")[0]
	ackArray := strings.Split(ackStr, ".")
	timestamp, _ := strconv.ParseInt(ackArray[1], 10, 64)
	rcvID := ackArray[2]

	if startTime != timestamp || peer.String() != rcvID {
		return
	}

	endTime := time.Now().UnixNano()
	rtt := endTime - startTime
	// Add to the table
	if rttTable[peer.String()] == nil {
		rttTable[peer.String()] = &RTTStruct{}
	}
	rttTable[peer.String()].totalRTT += rtt
	rttTable[peer.String()].samples++
	rttTable[peer.String()].avgRTT = float64(rttTable[peer.String()].totalRTT) / float64(rttTable[peer.String()].samples)
	fmt.Println("Received message: ", data)
	fmt.Println("Computer RTT (ns)", rtt)
	fmt.Printf("Average RTT for %s: %f\n", peer.String(), rttTable[peer.String()].avgRTT)
}
