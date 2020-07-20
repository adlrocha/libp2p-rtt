package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	mplex "github.com/libp2p/go-libp2p-mplex"
	secio "github.com/libp2p/go-libp2p-secio"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	tcp "github.com/libp2p/go-tcp-transport"
	"github.com/multiformats/go-multiaddr"
)

type mdnsNotifee struct {
	h   host.Host
	ctx context.Context
}

func (m *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	m.h.Connect(m.ctx, pi)
}

func main() {

	sourcePort := flag.Int("sp", 0, "Source port number")
	isBootstrap := flag.Bool("isbootstrap", false, "Display help")
	bootstrap := flag.String("b", "", "Bootstrap multiaddr string")
	debug := flag.Bool("debug", false, "Debug generates the same node ID on every execution")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure transports and muxers
	transports := libp2p.ChainOptions(
		libp2p.Transport(tcp.NewTCPTransport),
	)
	muxers := libp2p.ChainOptions(
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	)

	security := libp2p.Security(secio.ID, secio.New)

	var r io.Reader
	if *debug {
		r = mrand.New(mrand.NewSource(int64(1234)))
	} else {
		r = rand.Reader
	}

	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	id := libp2p.Identity(prvKey)

	listenAddrs := libp2p.ListenAddrStrings(
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *sourcePort),
	)

	var dht *kaddht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		dht, err = kaddht.New(ctx, h)
		return dht, err
	}
	routing := libp2p.Routing(newDHT)

	// Create host
	host, err := libp2p.New(
		ctx,
		transports,
		listenAddrs,
		muxers,
		security,
		routing,
		id,
	)
	if err != nil {
		panic(err)
	}

	host.SetStreamHandler("/rtt/0.0.1", func(s network.Stream) {
		RTTHandler(s, host)
	})

	for _, addr := range host.Addrs() {
		fmt.Printf("Listening on %s/p2p/%s\n", addr, host.ID().Pretty())
	}

	// Bootstrap node or connect bootstrap
	if !*isBootstrap {
		fmt.Println("Trying to connect to: ", *bootstrap)
		targetAddr, err := multiaddr.NewMultiaddr(*bootstrap)
		if err != nil {
			panic(err)
		}

		targetInfo, err := peer.AddrInfoFromP2pAddr(targetAddr)
		if err != nil {
			panic(err)
		}

		err = host.Connect(ctx, *targetInfo)
		if err != nil {
			panic(err)
		}

		fmt.Println("Connected to", targetInfo.ID)
	} else {
		fmt.Println("Starting bootstrap node...")
	}

	// Start discovery
	mdns, err := discovery.NewMdnsService(ctx, host, time.Second*10, "")
	if err != nil {
		panic(err)
	}
	mdns.RegisterNotifee(&mdnsNotifee{h: host, ctx: ctx})

	err = dht.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}

	// Start pingLoop
	if *isBootstrap {
		go sendPingLoop(host)
	}

	donec := make(chan struct{}, 1)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT)

	select {
	case <-stop:
		host.Close()
		os.Exit(0)
	case <-donec:
		host.Close()
	}
}
