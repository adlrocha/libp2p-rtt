# libp2p-rtt
Simple implementation of an RTT protocol in a libp2p network. Bootstrap nodes periodically
pings its connected nodes in order to track the average RTT with all the nodes it is
connected with. This simple protocol gives a local view of the links between bootstraps
and the node they are connected to (it a simple and unaccurate way of identifying the space
location of nodes in the network). Useful for different proof-of-concepts.

Under request I may maintain the project, meanwhile it have served me well. Do not hesitate to
contact me.

### Usage
* Build the code and use `--help` flag to see usage.
```
$ go build
$ rtt --help

Usage of ./rtt:
  -b string
        Bootstrap multiaddr string
  -debug
        Debug generates the same node ID on every execution
  -isbootstrap
        Display help
  -sp int
```
* To start a bootstrap node and start sending pings
```
$  ./rtt --isbootstrap
```

* To start a client node.
```
$ ./rtt -b <bootstrap_multiaddr>
```

