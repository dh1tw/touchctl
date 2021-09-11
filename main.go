package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	natsBroker "github.com/asim/go-micro/plugins/broker/nats/v3"
	natsReg "github.com/asim/go-micro/plugins/registry/nats/v3"
	natsTr "github.com/asim/go-micro/plugins/transport/nats/v3"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/transport"
	"github.com/dh1tw/remoteRotator/rotator"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/touchctl/hub"
	stackpage "github.com/dh1tw/touchctl/pages/stackmatch"
	nats "github.com/nats-io/nats.go"
	// profiling
	// _ "net/http/pprof"
)

const port int = 4222

func main() {

	urlFlag := flag.String("address", "localhost", "address of nats broker")
	usernameFlag := flag.String("username", "", "nats username")
	passwordFlag := flag.String("password", "", "nats password")

	flag.Parse()

	// Profiling (uncomment if needed)
	// go func() {
	// 	log.Println(http.ListenAndServe("0.0.0.0:6060", http.DefaultServeMux))
	// }()

	var reg registry.Registry
	var tr transport.Transport
	var br broker.Broker
	var cl client.Client

	username := *usernameFlag
	password := *passwordFlag
	url := *urlFlag

	if len(*usernameFlag) == 0 {
		log.Fatal("missing nats username")
	}
	if len(*passwordFlag) == 0 {
		log.Fatal("missing nats password")
	}

	connClosed := make(chan struct{})

	nopts := nats.GetDefaultOptions()
	nopts.Servers = []string{fmt.Sprintf("%s:%d", url, port)}
	nopts.User = username
	nopts.Password = password
	nopts.Timeout = time.Second * 10

	disconnectedHdlr := func(conn *nats.Conn) {
		log.Println("connection to nats broker closed")
		connClosed <- struct{}{}
	}
	// nopts.DisconnectedCB = disconnectHdlr

	errorHdlr := func(conn *nats.Conn, sub *nats.Subscription, err error) {
		log.Printf("Error Handler called (%s): %s", sub.Subject, err)
	}
	nopts.AsyncErrorCB = errorHdlr

	regNatsOpts := nopts
	brNatsOpts := nopts
	trNatsOpts := nopts
	regNatsOpts.DisconnectedCB = disconnectedHdlr
	regNatsOpts.Name = "touchCtl.client:registry"
	brNatsOpts.Name = "touchCtl.client:broker"
	trNatsOpts.Name = "touchCtl.client:transport"

	regTimeout := registry.Timeout(time.Second * 2)
	trTimeout := transport.Timeout(time.Second * 2)

	reg = natsReg.NewRegistry(natsReg.Options(regNatsOpts), regTimeout)
	tr = natsTr.NewTransport(natsTr.Options(trNatsOpts), trTimeout)
	br = natsBroker.NewBroker(natsBroker.Options(brNatsOpts))
	cl = client.NewClient(
		client.Broker(br),
		// client.Selector(static.NewSelector()),
		client.Transport(tr),
		client.Registry(reg),
		client.PoolSize(1),
		client.PoolTTL(time.Hour*8760), // one year - don't TTL our connection
		client.ContentType("application/proto-rpc"),
	)

	if err := cl.Init(); err != nil {
		log.Println(err)
		return
	}

	cache := &serviceCache{
		ttl:   time.Second * 20,
		cache: make(map[string]time.Time),
	}

	h, err := hub.NewHub()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	w := webserver{h, cl, cache}

	// watch the registry in a seperate thread for changes
	// at startup query the registry and add all found rotators
	if err := w.listAndAddServices(); err != nil {
		log.Println(err)
	}

	go w.watchRegistry()

	// Channel to handle OS signals
	osSignals := make(chan os.Signal, 1)

	//subscribe to os.Interrupt (CTRL-C signal)
	signal.Notify(osSignals, os.Interrupt)

	sd, err := esd.NewStreamDeck()
	if err != nil {
		log.Panic(err)
	}
	sd.ClearAllBtns()

	defer sd.ClearAllBtns()

	p := stackpage.NewStackPage(sd, nil, h)
	var pMutex sync.Mutex
	p.Draw()

	cb := func(keyIndex int, state esd.BtnState) {
		pMutex.Lock()
		defer pMutex.Unlock()
		newPage := p.Set(keyIndex, state)
		if newPage != nil {
			p = newPage
			sd.ClearAllBtns()
			p.Draw()
		}
	}

	sd.SetBtnEventCb(cb)

	select {
	case <-osSignals:
		return
	}
}

var bcast = make(chan rotator.Heading, 10)

var ev = func(r rotator.Rotator, status rotator.Heading) {
	//not used for the moment
}
