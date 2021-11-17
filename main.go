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

var myHub *hub.Hub

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

	myHub = h

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

	var bandSwAPage esd.Page
	var bandSwBPage esd.Page
	var stackSelectorPage esd.Page
	var stack10mPage esd.Page
	var stack15mPage esd.Page
	var stack20mPage esd.Page
	var stack40mPage esd.Page
	var stack80mPage esd.Page
	var stack160mPage esd.Page

	bandSwAConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "STCK", Position: 4, NextPage: &stackSelectorPage},
			genpage.Btn{Type: genpage.Label, Text: "A", Position: 9, NextPage: &bandSwBPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "160m", Text: "160m", Position: 8, NextPage: &stack160mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "80m", Text: "80m", Position: 7, NextPage: &stack80mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "40m", Text: "40m", Position: 6, NextPage: &stack40mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "20m", Text: "20m", Position: 5, NextPage: &stack20mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "15m", Text: "15m", Position: 13, NextPage: &stack15mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "10m", Text: "10m", Position: 12, NextPage: &stack10mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "6m", Text: "6m", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "A", TerminalName: "WARC", Text: "WARC", Position: 10},
		},
	}

	bandSwBConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "STCK", Position: 4, NextPage: &stackSelectorPage},
			genpage.Btn{Type: genpage.Label, Text: "B", Position: 9, NextPage: &bandSwAPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "160m", Text: "160m", Position: 8, NextPage: &stack160mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "80m", Text: "80m", Position: 7, NextPage: &stack80mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "40m", Text: "40m", Position: 6, NextPage: &stack40mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "20m", Text: "20m", Position: 5, NextPage: &stack20mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "15m", Text: "15m", Position: 13, NextPage: &stack15mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "10m", Text: "10m", Position: 12, NextPage: &stack10mPage},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "6m", Text: "6m", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Bandswitch", PortName: "B", TerminalName: "WARC", Text: "WARC", Position: 10},
		},
	}

	stackSelectorConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "BAND", NextPage: &bandSwAPage, Position: 4},
			genpage.Btn{Type: genpage.Label, Text: "160m", NextPage: &stack160mPage, Position: 8},
			genpage.Btn{Type: genpage.Label, Text: "80m", NextPage: &stack80mPage, Position: 7},
			genpage.Btn{Type: genpage.Label, Text: "40m", NextPage: &stack40mPage, Position: 6},
			genpage.Btn{Type: genpage.Label, Text: "20m", NextPage: &stack20mPage, Position: 5},
			genpage.Btn{Type: genpage.Label, Text: "15m", NextPage: &stack15mPage, Position: 13},
			genpage.Btn{Type: genpage.Label, Text: "10m", NextPage: &stack10mPage, Position: 12},
			genpage.Btn{Type: genpage.Label, Text: "N/A", Position: 11},
			genpage.Btn{Type: genpage.Label, Text: "N/A", Position: 10},
		},
	}

	sm10mConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "BAND", NextPage: &bandSwAPage, Position: 4},
			genpage.Btn{Type: genpage.Label, Text: "TWR4", Position: 0},
			genpage.Btn{Type: genpage.Label, Text: "TWR3", Position: 1},
			genpage.Btn{Type: genpage.Label, Text: "TWR2", Position: 2},
			genpage.Btn{Type: genpage.Label, Text: "TWR1", Position: 3},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower4", Position: 5},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower3", Position: 6},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower2", Position: 7},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower1", Position: 8},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 10m", PortName: "SM", TerminalName: "OB11-TWR3", Text: "OB11", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 10m", PortName: "SM", TerminalName: "OB11-TWR1", Text: "OB11", Position: 13},
		},
	}

	sm15mConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "BAND", NextPage: &bandSwAPage, Position: 4},
			genpage.Btn{Type: genpage.Label, Text: "TWR4", Position: 0},
			genpage.Btn{Type: genpage.Label, Text: "TWR3", Position: 1},
			genpage.Btn{Type: genpage.Label, Text: "TWR2", Position: 2},
			genpage.Btn{Type: genpage.Label, Text: "TWR1", Position: 3},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower4", Position: 5},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower3", Position: 6},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower2", Position: 7},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower1", Position: 8},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 15m", PortName: "SM", TerminalName: "4L-TWR4", Text: "4L", Position: 10},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 15m", PortName: "SM", TerminalName: "OB11-TWR3", Text: "OB11", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 15m", PortName: "SM", TerminalName: "OB11-TWR1", Text: "OB11", Position: 13},
		},
	}

	sm20mConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "BAND", NextPage: &bandSwAPage, Position: 4},
			genpage.Btn{Type: genpage.Label, Text: "TWR4", Position: 0},
			genpage.Btn{Type: genpage.Label, Text: "TWR3", Position: 1},
			genpage.Btn{Type: genpage.Label, Text: "TWR2", Position: 2},
			genpage.Btn{Type: genpage.Label, Text: "TWR1", Position: 3},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower4", Position: 5},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower3", Position: 6},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower2", Position: 7},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower1", Position: 8},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 20m", PortName: "SM", TerminalName: "OB11-TWR3", Text: "OB11", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 20m", PortName: "SM", TerminalName: "OB11-TWR2", Text: "OB11", Position: 12},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 20m", PortName: "SM", TerminalName: "OB11-TWR1", Text: "OB11", Position: 13},
		},
	}

	sm40mConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "BAND", NextPage: &bandSwAPage, Position: 4},
			genpage.Btn{Type: genpage.Label, Text: "TWR4", Position: 0},
			genpage.Btn{Type: genpage.Label, Text: "TWR3", Position: 1},
			genpage.Btn{Type: genpage.Label, Text: "TWR2", Position: 2},
			genpage.Btn{Type: genpage.Label, Text: "TWR1", Position: 3},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower4", Position: 5},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower3", Position: 6},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower2", Position: 7},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower1", Position: 8},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 40m", PortName: "SM", TerminalName: "DIPOL-TWR3", Text: "DIPL", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 40m", PortName: "SM", TerminalName: "2L-TWR1", Text: "2L", Position: 13},
		},
	}
	sm80mConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "BAND", NextPage: &bandSwAPage, Position: 4},
			genpage.Btn{Type: genpage.Label, Text: "TWR4", Position: 0},
			genpage.Btn{Type: genpage.Label, Text: "TWR3", Position: 1},
			genpage.Btn{Type: genpage.Label, Text: "TWR2", Position: 2},
			genpage.Btn{Type: genpage.Label, Text: "TWR1", Position: 3},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower4", Position: 5},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower3", Position: 6},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower2", Position: 7},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower1", Position: 8},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 80m", PortName: "SM", TerminalName: "VERTICAL", Text: "VERT", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 80m", PortName: "SM", TerminalName: "DIPOL", Text: "DIPL", Position: 13},
		},
	}
	sm160mConfig := genpage.Config{
		Layout: []genpage.Btn{
			genpage.Btn{Type: genpage.Label, Text: "BAND", NextPage: &bandSwAPage, Position: 4},
			genpage.Btn{Type: genpage.Label, Text: "TWR4", Position: 0},
			genpage.Btn{Type: genpage.Label, Text: "TWR3", Position: 1},
			genpage.Btn{Type: genpage.Label, Text: "TWR2", Position: 2},
			genpage.Btn{Type: genpage.Label, Text: "TWR1", Position: 3},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower4", Position: 5},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower3", Position: 6},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower2", Position: 7},
			genpage.Btn{Type: genpage.Rotator, DeviceName: "Tower1", Position: 8},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 160m", PortName: "SM", TerminalName: "VERTICAL", Text: "VERT", Position: 11},
			genpage.Btn{Type: genpage.Terminal, DeviceName: "Stackmatch 160m", PortName: "SM", TerminalName: "DIPOL", Text: "DIPL", Position: 13},
		},
	}

	bandSwA, err := genpage.NewGenPage(sd, h, bandSwAConfig)
	if err != nil {
		log.Panic(err)
	}
	bandSwAPage = bandSwA

	bandSwB, err := genpage.NewGenPage(sd, h, bandSwBConfig)
	if err != nil {
		log.Panic(err)
	}
	bandSwBPage = bandSwB

	stackSelector, err := genpage.NewGenPage(sd, h, stackSelectorConfig)
	if err != nil {
		log.Panic(err)
	}
	stackSelectorPage = stackSelector

	stack10m, err := genpage.NewGenPage(sd, h, sm10mConfig)
	if err != nil {
		log.Panic(err)
	}
	stack10mPage = stack10m

	stack15m, err := genpage.NewGenPage(sd, h, sm15mConfig)
	if err != nil {
		log.Panic(err)
	}
	stack15mPage = stack15m

	stack20m, err := genpage.NewGenPage(sd, h, sm20mConfig)
	if err != nil {
		log.Panic(err)
	}
	stack20mPage = stack20m

	stack40m, err := genpage.NewGenPage(sd, h, sm40mConfig)
	if err != nil {
		log.Panic(err)
	}
	stack40mPage = stack40m

	stack80m, err := genpage.NewGenPage(sd, h, sm80mConfig)
	if err != nil {
		log.Panic(err)
	}
	stack80mPage = stack80m

	stack160m, err := genpage.NewGenPage(sd, h, sm160mConfig)
	if err != nil {
		log.Panic(err)
	}
	stack160mPage = stack160m

	h.SubscribeToSbDeviceStatus("BandSwitchA", bandSwA.HandleSbDeviceUpdate)
	h.SubscribeToSbDeviceStatus("BandSwitchB", bandSwB.HandleSbDeviceUpdate)
	h.SubscribeToSbDeviceStatus("Stackmatch10m", stack10m.HandleSbDeviceUpdate)
	h.SubscribeToSbDeviceStatus("Stackmatch15m", stack15m.HandleSbDeviceUpdate)
	h.SubscribeToSbDeviceStatus("Stackmatch20m", stack20m.HandleSbDeviceUpdate)
	h.SubscribeToSbDeviceStatus("Stackmatch40m", stack40m.HandleSbDeviceUpdate)
	h.SubscribeToSbDeviceStatus("Stackmatch80m", stack80m.HandleSbDeviceUpdate)
	h.SubscribeToSbDeviceStatus("Stackmatch160m", stack160m.HandleSbDeviceUpdate)

	var currentPage esd.Page
	currentPage = bandSwAPage

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

	// at startup, query the registry and add all found rotators and switches
	// if err := w.listAndAddServices(); err != nil {
	// 	log.Println(err)
	// }

	// watch the registry in a seperate thread for changes
	go w.watchRegistry()

	<-osSignals
}

var rotatorEvent = func(r rotator.Rotator, status rotator.Heading) {
	ev := hub.SbDeviceStatusEvent{
		Event:      hub.SbUpdateDevice,
		DeviceName: r.Name(),
	}
	myHub.BroadcastSbDeviceStatus(ev)
}

var switchEvent = func(s sw.Switcher, device sw.Device) {
	ev := hub.SbDeviceStatusEvent{
		Event:      hub.SbUpdateDevice,
		DeviceName: s.Name(),
	}
	myHub.BroadcastSbDeviceStatus(ev)
}
