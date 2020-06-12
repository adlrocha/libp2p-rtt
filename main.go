package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"os"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"

	"github.com/multiformats/go-multiaddr"
)

// func PingHandle(req Request) {
// 	res := &proto.Request{
// 		Type: proto.Request_ACK_MESSAGE,
// 		AckMessage: &proto.AckMessage{
// 			code: 1
// 		}
// 	}
// 	MsgSend(req,)

// }

// // TODO: Extract to the node. Structure code.
// func SendProtoMsg(n *Node, id peer.ID, data proto.Message) {
// 	s, err := n.hosts.NewStream(context.Background(), id, protocol.VtnProtocol)
// 	if err != nil {
// 		log.Println(err)
// 		return false
// 	}
// 	writer := ggio.NewFullWriter(s)
// 	err = writer.WriteMsg(data)
// 	if err != nil {
// 		log.Println(err)
// 		s.Reset()
// 		return false
// 	}
// 	// FullClose closes the stream and waits for the other side to close their half.
// 	err = helpers.FullClose(s)
// 	if err != nil {
// 		log.Println(err)
// 		s.Reset()
// 		return false
// 	}
// 	return true
// }

// func MsgHandler(s network.Stream) {
// 	data, err := ioutil.ReadAll(s)
// 	if err != nil {
// 		fmt.Fprintln(os.Stderr, err)
// 	}

// 	rcv := Request{}
// 	proto.Unmarshal(data, rcv)
// 	fmt.Println("Stream multiaddress", s.Conn().RemoteMultiaddr.String())
// 	switch *req.Type {
// 	case Request_PING_MESSAGE:
// 		PingHandler(rcv)
// 	defualt:
// 		fmt.Println("Received wrong message...")
// 	}
// }

// func MsgSend(req Request, s network.Stream) error {
// 	msgBytes, err := proto.Marshal(req)
// 	return error
// }

func handleStream(s network.Stream) {
	log.Println("Got a new stream!")
	if err := handleMsg(s); err != nil {
		log.Println(err)
		s.Reset()
	} else {
		s.Close()
	}
}
func handleMsg(s network.Stream) error {
	// // Create a buffer stream for non blocking read and write.
	// rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	// str, _ := rw.ReadString('\n')

	// if str == "" {
	// 	return nil
	// }
	// if str != "\n" {
	// 	// Green console colour: 	\x1b[32m
	// 	// Reset console colour: 	\x1b[0m
	// 	fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
	// }
	// // TODO: This is key!!! Start from here the RPC
	// log.Println("Received", str)

	// fmt.Println("Sending ACK...")
	// if str != "ACK\n" {
	// 	rw.Flush()
	// 	rw.WriteString("ACK\n")
	// }
	// s.Close()
	// go readData(rw)
	// go writeData(rw)

	// stream 's' will stay open until you close it (or the other side closes it).
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	log.Printf("read: %s\n", str)
	fmt.Println()
	_, err = s.Write([]byte("ACK\n"))
	return err
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}
		// TODO: This is key!!! Start from here the RPC
		log.Println("Received", str)

		if str != "ACK\n" {
			rw.Flush()
			rw.WriteString("ACK\n")
		}

	}
}

func sendRTT(host host.Host, id peer.ID) {

	for {
		// fmt.Print("> ")
		// sendData, err := stdReader.ReadString('\n')

		// if err != nil {
		// 	panic(err)
		// }
		s, err := host.NewStream(context.Background(), id, "/chat/1.0.0")
		if err != nil {
			panic(err)
		}
		// defer s.Close()
		// Create a buffered stream so that read and writes are non blocking.
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		sendData := "ping" + strconv.Itoa(int(time.Now().Unix()))
		rw.WriteString(fmt.Sprintf("%s\n", sendData))
		rw.Flush()
		time.Sleep(5 * time.Second)
	}

}

func main() {
	sourcePort := flag.Int("sp", 0, "Source port number")
	dest := flag.String("d", "", "Destination multiaddr string")
	help := flag.Bool("help", false, "Display help")
	debug := flag.Bool("debug", false, "Debug generates the same node ID on every execution")

	flag.Parse()

	if *help {
		fmt.Printf("This program demonstrates a simple p2p chat application using libp2p\n\n")
		fmt.Println("Usage: Run './chat -sp <SOURCE_PORT>' where <SOURCE_PORT> can be any port number.")
		fmt.Println("Now run './chat -d <MULTIADDR>' where <MULTIADDR> is multiaddress of previous listener host.")

		os.Exit(0)
	}

	// If debug is enabled, use a constant random source to generate the peer ID. Only useful for debugging,
	// off by default. Otherwise, it uses rand.Reader.
	var r io.Reader
	if *debug {
		// Use the port number as the randomness source.
		// This will always generate the same host ID on multiple executions, if the same port number is used.
		// Never do this in production code.
		r = mrand.New(mrand.NewSource(int64(*sourcePort)))
	} else {
		r = rand.Reader
	}

	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *sourcePort))

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)

	if err != nil {
		panic(err)
	}

	if *dest == "" {
		// Set a function as stream handler.
		// This function is called when a peer connects, and starts a stream with this protocol.
		// Only applies on the receiving side.
		host.SetStreamHandler("/chat/1.0.0", handleStream)

		// Let's get the actual TCP port from our listen multiaddr, in case we're using 0 (default; random available port).
		var port string
		for _, la := range host.Network().ListenAddresses() {
			if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
				port = p
				break
			}
		}

		if port == "" {
			panic("was not able to find actual local port")
		}

		fmt.Printf("Run './chat -d /ip4/127.0.0.1/tcp/%v/p2p/%s' on another console.\n", port, host.ID().Pretty())
		fmt.Println("You can replace 127.0.0.1 with public IP as well.")
		fmt.Printf("\nWaiting for incoming connection\n\n")

		// Hang forever
		<-make(chan struct{})
	} else {
		fmt.Println("This node's multiaddresses:")
		for _, la := range host.Addrs() {
			fmt.Printf(" - %v\n", la)
		}
		fmt.Println()

		// Turn the destination into a multiaddr.
		maddr, err := multiaddr.NewMultiaddr(*dest)
		if err != nil {
			log.Fatalln(err)
		}

		// Extract the peer ID from the multiaddr.
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Fatalln(err)
		}

		// Add the destination's peer multiaddress in the peerstore.
		// This will be used during connection and stream creation by libp2p.
		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

		// Start a stream with the destination.
		// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
		// s, err := host.NewStream(context.Background(), info.ID, "/chat/1.0.0")
		// if err != nil {
		// 	panic(err)
		// }

		// Create a buffered stream so that read and writes are non blocking.
		// rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		// Create a thread to read and write data.
		go sendRTT(host, info.ID)
		// go readData(rw)

		// Hang forever.
		select {}
	}
}
