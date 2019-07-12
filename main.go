package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/dh1tw/remoteRotator/rotator"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/touchctl/hub"
	stackpage "github.com/dh1tw/touchctl/pages/stackmatch"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector/static"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
	natsBroker "github.com/micro/go-plugins/broker/nats"
	natsReg "github.com/micro/go-plugins/registry/nats"
	natsTr "github.com/micro/go-plugins/transport/nats"
	nats "github.com/nats-io/nats.go"
)

const url string = "ed1r.ddns.net"
const port int = 4222

func main() {

	usernameFlag := flag.String("username", "", "nats username")
	passwordFlag := flag.String("password", "", "nats password")

	flag.Parse()

	var reg registry.Registry
	var tr transport.Transport
	var br broker.Broker
	var cl client.Client

	username := *usernameFlag
	password := *passwordFlag

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
		client.Selector(static.NewSelector()),
		client.Transport(tr),
		client.Registry(reg),
		client.PoolSize(1),
		client.PoolTTL(time.Hour*8760), // one year - don't TTL our connection
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
	p.Draw()

	cb := func(keyIndex int, state esd.BtnState) {
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
	bcast <- status
}
