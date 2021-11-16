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
	sw "github.com/dh1tw/remoteSwitch/switch"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/touchctl/hub"
	genpage "github.com/dh1tw/touchctl/pages/genPage"
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

	// smConfig10m := stackpage.StackConfig{
	// 	Band: "10m",
	// 	Name: "Stackmatch 10m",
	// 	Ant1: stackpage.SmTerminal{Name: "OB11-TWR1", ShortName: "OB11", Index: 0},
	// 	Ant2: stackpage.SmTerminal{Name: "OB11-TWR2", ShortName: "OB11", Index: 1},
	// 	Ant3: stackpage.SmTerminal{Name: "OB11-TWR3", ShortName: "OB11", Index: 2},
	// }

	// smConfig15m := stackpage.StackConfig{
	// 	Band: "15m",
	// 	Name: "Stackmatch 15m",
	// 	Ant1: stackpage.SmTerminal{Name: "OB11-TWR1", ShortName: "OB11", Index: 0},
	// 	Ant3: stackpage.SmTerminal{Name: "OB11-TWR3", ShortName: "OB11", Index: 2},
	// 	Ant4: stackpage.SmTerminal{Name: "4L-TWR4", ShortName: " 4L ", Index: 3},
	// }

	// smConfig20m := stackpage.StackConfig{
	// 	Band: "20m",
	// 	Name: "Stackmatch 20m",
	// 	Ant1: stackpage.SmTerminal{Name: "OB11-TWR1", ShortName: "OB11", Index: 0},
	// 	Ant2: stackpage.SmTerminal{Name: "OB11-TWR2", ShortName: "OB11", Index: 1},
	// 	Ant3: stackpage.SmTerminal{Name: "OB11-TWR3", ShortName: "OB11", Index: 2},
	// }

	// smConfig40m := stackpage.StackConfig{
	// 	Band: "40m",
	// 	Name: "Stackmatch 40m",
	// 	Ant1: stackpage.SmTerminal{Name: "2L-TWR1", ShortName: " 2L ", Index: 0},
	// 	Ant3: stackpage.SmTerminal{Name: "DIPOL-TWR3", ShortName: "DIPL", Index: 2},
	// }

	// p10m := stackpage.NewStackPage(sd, nil, h, smConfig10m)
	// p15m := stackpage.NewStackPage(sd, nil, h, smConfig15m)
	// p20m := stackpage.NewStackPage(sd, nil, h, smConfig20m)
	// p40m := stackpage.NewStackPage(sd, nil, h, smConfig40m)

	// rotatorEvents["p10m"] = p10m.RotatorUpdateHandler
	// rotatorEvents["p15m"] = p15m.RotatorUpdateHandler
	// rotatorEvents["p20m"] = p20m.RotatorUpdateHandler
	// rotatorEvents["p40m"] = p40m.RotatorUpdateHandler

	// switchEvents["p10m"] = p10m.SwitchUpdateHandler
	// switchEvents["p15m"] = p15m.SwitchUpdateHandler
	// switchEvents["p20m"] = p20m.SwitchUpdateHandler
	// switchEvents["p40m"] = p40m.SwitchUpdateHandler

	// h.SubscribeToSbDeviceStatus("p10m", p10m.SbDeviceStatusHandler)

	// stacks := map[string]esd.Page{
	// 	"10m": p10m,
	// 	"15m": p15m,
	// 	"20m": p20m,
	// 	"40m": p40m,
	// }

	gpConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "TWR4", Position: 0},
			genpage.Btn{Type: genpage.Label, Text: "TWR3", Position: 1},
			genpage.Btn{Type: genpage.Label, Text: "TWR2", Position: 2},
			genpage.Btn{Type: genpage.Label, Text: "TWR1", Position: 3},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower4", Position: 5},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower3", Position: 6},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower2", Position: 7},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower1", Position: 8},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 20m", ItemName: "OB11-TWR3", Text: "OB11", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 20m", ItemName: "OB11-TWR2", Text: "OB11", Position: 12},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 20m", ItemName: "OB11-TWR1", Text: "OB11", Position: 13},
		},
	}

	gp, err := genpage.NewGenPage(sd, h, gpConfig)
	if err != nil {
		log.Panic(err)
	}

	h.SubscribeToSbDeviceStatus("20m", gp.HandleSbDeviceUpdate)

	var currentPage esd.Page
	currentPage = gp

	// currentPage := bandpage.NewBandPage(sd, nil, stacks)
	// p10m.SetParent(currentPage)
	// p15m.SetParent(currentPage)
	// p20m.SetParent(currentPage)
	// p40m.SetParent(currentPage)

	var pMutex sync.Mutex
	currentPage.SetActive(true)
	currentPage.Draw()

	cb := func(keyIndex int, state esd.BtnState) {
		pMutex.Lock()
		defer pMutex.Unlock()
		newPage := currentPage.Set(keyIndex, state)
		if newPage != nil {
			currentPage.SetActive(false)
			sd.ClearAllBtns()
			currentPage = newPage
			currentPage.SetActive(true)
			currentPage.Draw()
		}
	}

	sd.SetBtnEventCb(cb)

	// // at startup, query the registry and add all found rotators and switches
	// if err := w.listAndAddServices(); err != nil {
	// 	log.Println(err)
	// }

	// watch the registry in a seperate thread for changes
	go w.watchRegistry()

	select {
	case <-osSignals:
		return
	}
}

var rotatorEvents map[string]func(r rotator.Rotator, status rotator.Heading) = map[string]func(r rotator.Rotator, status rotator.Heading){}
var switchEvents map[string]func(s sw.Switcher, device sw.Device) = map[string]func(s sw.Switcher, device sw.Device){}

var rotatorEvent = func(r rotator.Rotator, status rotator.Heading) {
	// fmt.Printf("rotor event: %v %v°\n", r.Name(), r.Azimuth())
	for _, handler := range rotatorEvents {
		go handler(r, status)
	}
}

var switchEvent = func(s sw.Switcher, device sw.Device) {
	// fmt.Println("switch event: ", device)
	for _, handler := range switchEvents {
		go handler(s, device)
	}
}
