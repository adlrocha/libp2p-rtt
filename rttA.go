package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mrand "math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"

	ma "github.com/multiformats/go-multiaddr"
)

// TODO: add go-libp2p-examples
//  ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	transports := libp2p.ChainOptions(
// 		libp2p.Transport(tcp.NewTCPTransport),
// 		libp2p.Transport(ws.New),
// 	)

// 	muxers := libp2p.ChainOptions(
// 		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
// 		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
// 	)

// 	security := libp2p.Security(secio.ID, secio.New)

// 	listenAddrs := libp2p.ListenAddrStrings(
// 		"/ip4/0.0.0.0/tcp/0",
// 		"/ip4/0.0.0.0/tcp/0/ws",
// 	)

// 	var dht *kaddht.IpfsDHT
// 	newDHT := func(h host.Host) (routing.PeerRouting, error) {
// 		var err error
// 		dht, err = kaddht.New(ctx, h)
// 		return dht, err
// 	}
// 	routing := libp2p.Routing(newDHT)

// 	host, err := libp2p.New(
// 		ctx,
// 		transports,
// 		listenAddrs,
// 		muxers,
// 		security,
// 		routing,
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	ps, err := pubsub.NewGossipSub(ctx, host)
// 	if err != nil {
// 		panic(err)
// 	}
// 	sub, err := ps.Subscribe(pubsubTopic)
// 	if err != nil {
// 		panic(err)
// 	}
// 	go pubsubHandler(ctx, sub)

// 	for _, addr := range host.Addrs() {
// 		fmt.Println("Listening on", addr)
// 	}

// 	targetAddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/63785/p2p/QmWjz6xb8v9K4KnYEwP5Yk75k5mMBCehzWFLCvvQpYxF3d")
// 	if err != nil {
// 		panic(err)
// 	}

// 	targetInfo, err := peer.AddrInfoFromP2pAddr(targetAddr)
// 	if err != nil {
// 		panic(err)
// 	}

// 	err = host.Connect(ctx, *targetInfo)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("Connected to", targetInfo.ID)

// 	mdns, err := discovery.NewMdnsService(ctx, host, time.Second*10, "")
// 	if err != nil {
// 		panic(err)
// 	}
// 	mdns.RegisterNotifee(&mdnsNotifee{h: host, ctx: ctx})

// 	err = dht.Bootstrap(ctx)
// 	if err != nil {
// 		panic(err)
// 	}

// 	donec := make(chan struct{}, 1)
// 	go chatInputLoop(ctx, host, ps, donec)

// 	stop := make(chan os.Signal, 1)
// 	signal.Notify(stop, syscall.SIGINT)

// 	select {
// 	case <-stop:
// 		host.Close()
// 		os.Exit(0)
// 	case <-donec:
// 		host.Close()
// 	}

// makeBasicHost creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It won't encrypt the connection if insecure is true.
func makeBasicHost(listenPort int, insecure bool, randseed int64) (host.Host, error) {

	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}

	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	log.Printf("I am %s\n", fullAddr)
	if insecure {
		log.Printf("Now run \"./echo -l %d -d %s -insecure\" on a different terminal\n", listenPort+1, fullAddr)
	} else {
		log.Printf("Now run \"./echo -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddr)
	}

	return basicHost, nil
}

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	// golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	insecure := flag.Bool("insecure", false, "use an unencrypted connection")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	// Make a host that listens on the given multiaddress
	ha, err := makeBasicHost(*listenF, *insecure, *seed)
	if err != nil {
		log.Fatal(err)
	}

	// Set a stream handler on host A. /echo/1.0.0 is
	// a user-defined protocol name.
	ha.SetStreamHandler("/echo/1.0.0", func(s network.Stream) {
		log.Println("Got a new stream!")
		if err := doEcho(s); err != nil {
			log.Println(err)
			s.Reset()
		} else {
			s.Close()
		}
	})

	if *target == "" {
		log.Println("listening for connections")
		select {} // hang forever
	}
	/**** This is where the listener code ends ****/

	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(*target)
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", pid))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	ha.Peerstore().AddAddr(peerid, targetAddr, peerstore.PermanentAddrTTL)

	log.Println("opening stream")
	// make a new stream from host B to host A
	// it should be handled on host A by the handler we set above because
	// we use the same /echo/1.0.0 protocol

	// TODO: Implement here a RTT computation.
	for {
		startTime := time.Now().UnixNano()
		s, err := ha.NewStream(context.Background(), peerid, "/echo/1.0.0")
		if err != nil {
			log.Fatalln(err)
		}

		_, err = s.Write([]byte(fmt.Sprintf("%s.%s\n", "ping", strconv.FormatInt(startTime, 10))))
		if err != nil {
			log.Fatalln(err)
		}

		out, err := ioutil.ReadAll(s)
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("read reply: %q\n", out)
		ackTimeStr := strings.Split(string(out), "\n")[0]
		ackTimeStr = strings.Split(ackTimeStr, ".")[1]
		ackTime, _ := strconv.ParseInt(ackTimeStr, 10, 64)
		fmt.Println(ackTime)
		fmt.Println("RTT time (ns): ", ackTime-startTime)
		time.Sleep(5 * time.Second)
	}
}

// doEcho reads a line of data a stream and writes it back
func doEcho(s network.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	log.Printf("read: %s\n", str)
	_, err = s.Write([]byte(fmt.Sprintf("%s.%s\n", "ACK", strconv.FormatInt(time.Now().UnixNano(), 10))))
	return err
}
