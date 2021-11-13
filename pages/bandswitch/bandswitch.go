package bandswitch

import (
	"log"
	"sync"

	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck-buttons/label"
	ledBtn "github.com/dh1tw/streamdeck-buttons/ledbutton"
	"github.com/dh1tw/touchctl/hub"
)

type BandswitchPage struct {
	sd *esd.StreamDeck
	sync.Mutex
	ownParent  esd.Page
	labels     map[int]*label.Label
	hub        *hub.Hub
	active     bool
	activePort string
	config     BandswitchConfig
	portABtns  map[int]*ledBtn.LedButton
	portBBtns  map[int]*ledBtn.LedButton
}

type bandswitchPort struct {
	sync.Mutex
}

type BandswitchConfig struct {
	Name              string
	BandButtonMapping map[string]int
}

func NewBandswitchPage(sd *esd.StreamDeck, parent esd.Page, h *hub.Hub, config BandswitchConfig) *BandswitchPage {

	bsp := &BandswitchPage{
		sd:         sd,
		ownParent:  parent,
		hub:        h,
		config:     config,
		activePort: "A",
	}

	bs, exists := bsp.hub.Switch(bsp.config.Name)
	if !exists {
		log.Fatalf("bandswitch '%v' does not exist", bsp.config.Name)
	}

	portA, err := bs.GetPort("A")
	if err != nil {
		log.Fatalf("port A on bandswitch '%v' does not exist", bsp.config.Name)
	}

	// portB, err := bs.GetPort("B")
	// if err != nil {
	// 	log.Fatalf("port B on bandswitch '%v' does not exist", bsp.config.Name)
	// }

	for _, t := range portA.Terminals {
		if pos, exists := bsp.config.BandButtonMapping[t.Name]; exists {
			b, err := ledBtn.NewLedButton(sd, 13, ledBtn.Text(t.Name))
			if err != nil {
				log.Fatal(err)
			}
			b.SetState(t.State)
			bsp.portABtns[pos] = b
		}
	}

	return bsp
}
