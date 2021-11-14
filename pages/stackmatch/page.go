package stackpage

import (
	"fmt"
	"image/color"
	"log"
	"sort"
	"sync"

	Switch "github.com/dh1tw/remoteSwitch/switch"

	"github.com/dh1tw/remoteRotator/rotator"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck-buttons/label"
	ledBtn "github.com/dh1tw/streamdeck-buttons/ledbutton"
	"github.com/dh1tw/touchctl/hub"
	rotatorpage "github.com/dh1tw/touchctl/pages/rotator"
)

type StackPage struct {
	sd *esd.StreamDeck
	sync.Mutex
	ownParent esd.Page
	stack     *stackmatch
	rotators  map[int]*rot
	labels    map[int]*label.Label
	hub       *hub.Hub
	active    bool
	config    StackConfig
}

type rot struct {
	rotator rotator.Rotator
	label   *label.Label
}

type stackmatch struct {
	sync.Mutex
	sm   Switch.Switcher
	btns map[string]*ledBtn.LedButton
}

func (sm *stackmatch) set(terminalName string) error {
	sm.Lock()
	defer sm.Unlock()
	t, ok := sm.btns[terminalName]
	if !ok {
		return fmt.Errorf("unknown terminal %s", terminalName)
	}

	p := Switch.Port{
		Name: "SM",
		Terminals: []Switch.Terminal{
			Switch.Terminal{
				Name:  terminalName,
				State: !t.State(),
			},
		},
	}

	err := sm.sm.SetPort(p)
	if err != nil {
		return err
	}

	return nil
}

type SmTerminal struct {
	Name      string // Full name
	ShortName string // max 4 char
	Index     int
}

type StackConfig struct {
	Band string
	Name string
	Ant1 SmTerminal
	Ant2 SmTerminal
	Ant3 SmTerminal
	Ant4 SmTerminal
}

func NewStackPage(sd *esd.StreamDeck, parent esd.Page, h *hub.Hub, smConfig StackConfig) *StackPage {

	sp := &StackPage{
		sd:        sd,
		ownParent: parent,
		rotators:  make(map[int]*rot, 0),
		labels:    make(map[int]*label.Label),
		hub:       h,
		config:    smConfig,
		active:    false,
	}

	labels := map[int]string{
		3: "TWR1",
		2: "TWR2",
		1: "TWR3",
		0: "TWR4",
	}

	for pos, txt := range labels {
		tb, err := label.NewLabel(sd, pos, label.Text(txt), label.TextColor(color.RGBA{92, 184, 92, 255}))
		if err != nil {
			log.Fatal(err)
		}
		sp.labels[pos] = tb
	}

	bandLabel, err := label.NewLabel(sd, 14, label.Text(smConfig.Band), label.TextColor(color.RGBA{255, 0, 0, 255}))
	if err != nil {
		log.Fatal(err)
	}

	sp.labels[14] = bandLabel

	rots := sp.hub.Rotators()
	sort.Slice(rots, func(i, j int) bool {
		return rots[i].Name() < rots[j].Name()
	})

	stack, exists := sp.hub.Switch(smConfig.Name)
	if !exists {
		log.Fatalf("%v doesn't exist", smConfig.Name)
	}

	port, err := stack.GetPort("SM")
	if err != nil {
		log.Fatalf("port SM on %v doesn't exist", smConfig.Name)
	}

	sm := &stackmatch{
		sm:   stack,
		btns: make(map[string]*ledBtn.LedButton),
	}

	for _, t := range port.Terminals {

		switch t.Name {
		case smConfig.Ant1.Name:
			b, err := ledBtn.NewLedButton(sd, 13, ledBtn.Text(smConfig.Ant1.ShortName), ledBtn.State(t.State))
			if err != nil {
				log.Fatal(err)
			}
			sm.btns[smConfig.Ant1.Name] = b
		case smConfig.Ant2.Name:
			b, err := ledBtn.NewLedButton(sd, 12, ledBtn.Text(smConfig.Ant2.ShortName), ledBtn.State(t.State))
			if err != nil {
				log.Fatal(err)
			}
			sm.btns[smConfig.Ant2.Name] = b
		case smConfig.Ant3.Name:
			b, err := ledBtn.NewLedButton(sd, 11, ledBtn.Text(smConfig.Ant3.ShortName), ledBtn.State(t.State))
			if err != nil {
				log.Fatal(err)
			}
			sm.btns[smConfig.Ant3.Name] = b
		case smConfig.Ant4.Name:
			b, err := ledBtn.NewLedButton(sd, 10, ledBtn.Text(smConfig.Ant4.ShortName), ledBtn.State(t.State))
			if err != nil {
				log.Fatal(err)
			}
			sm.btns[smConfig.Ant4.Name] = b
		}
	}

	sp.stack = sm

	counter := 8
	for _, r := range rots {
		fmt.Printf("%v - %v: %03d°\n", sp.config.Band, r.Name(), r.Azimuth())
		lbl, err := label.NewLabel(sd, counter, label.Text(fmt.Sprintf("%03d°", r.Azimuth())))
		if err != nil {
			log.Panic(err)
		}
		r := &rot{
			rotator: r,
			label:   lbl,
		}

		sp.rotators[counter] = r
		counter--
	}

	return sp
}

func (sp *StackPage) SbDeviceStatusHandler(event hub.SbDeviceStatusEvent) {
	log.Println(event)
}

func (sp *StackPage) Set(btnIndex int, state esd.BtnState) esd.Page {

	sp.Lock()
	defer sp.Unlock()

	if state == esd.BtnReleased {
		return nil
	}

	switch btnIndex {
	case 14:
		if sp.ownParent == nil {
			return nil
		}
		return sp.ownParent
	case 10:
		if err := sp.stack.set(sp.config.Ant4.Name); err != nil {
			log.Println(err)
		}
	case 11:
		if err := sp.stack.set(sp.config.Ant3.Name); err != nil {
			log.Println(err)
		}
	case 12:
		if err := sp.stack.set(sp.config.Ant2.Name); err != nil {
			log.Println(err)
		}
	case 13:
		if err := sp.stack.set(sp.config.Ant1.Name); err != nil {
			log.Println(err)
		}
	default: // rotator
		rot, ok := sp.rotators[btnIndex]
		if ok {
			return rotatorpage.NewRotatorPage(sp.sd, sp, rot.rotator)
		}
	}

	return nil
}

func (sp *StackPage) RotatorUpdateHandler(r rotator.Rotator, status rotator.Heading) {
	var rLabel *rot
	sp.Lock()
	defer sp.Unlock()
	switch r.Name() {
	case "Tower1":
		rLabel = sp.rotators[8]
	case "Tower2":
		rLabel = sp.rotators[7]
	case "Tower3":
		rLabel = sp.rotators[6]
	case "Tower4":
		rLabel = sp.rotators[5]
	}
	if rLabel != nil {
		rLabel.label.SetText(fmt.Sprintf("%03d°", r.Azimuth()))
		if sp.active {
			rLabel.label.Draw()
		}
	}
}

func (sp *StackPage) SwitchUpdateHandler(s Switch.Switcher, device Switch.Device) {
	sp.Lock()
	defer sp.Unlock()
	if device.Name != sp.config.Name {
		return
	}
	p, err := s.GetPort("SM")
	if err != nil {
		log.Println(err)
		return
	}
	sp.stack.Lock()
	defer sp.stack.Unlock()
	for _, t := range p.Terminals {
		btn, ok := sp.stack.btns[t.Name]
		if !ok {
			log.Printf("unknown terminal %s", t.Name)
		}
		btn.SetState(t.State)
		if sp.active {
			btn.Draw()
		}
	}
}

func (sp *StackPage) SetActive(active bool) {
	sp.Lock()
	defer sp.Unlock()
	sp.active = active
}

func (sp *StackPage) draw() {
	for _, label := range sp.labels {
		label.Draw()
	}

	for _, rot := range sp.rotators {
		rot.label.Draw()
	}

	for _, btn := range sp.stack.btns {
		btn.Draw()
	}
}

func (sp *StackPage) Draw() {
	sp.Lock()
	defer sp.Unlock()
	sp.draw()
}

func (sp *StackPage) Parent() esd.Page {
	sp.Lock()
	defer sp.Unlock()
	return sp.ownParent
}

func (sp *StackPage) SetParent(parent esd.Page) {
	sp.Lock()
	defer sp.Unlock()
	sp.ownParent = parent
}
