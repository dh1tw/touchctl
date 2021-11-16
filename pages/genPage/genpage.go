package genpage

import (
	"fmt"
	"image/color"
	"log"
	"sync"

	"github.com/dh1tw/remoteRotator/rotator"
	sw "github.com/dh1tw/remoteSwitch/switch"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck-buttons/label"
	ledBtn "github.com/dh1tw/streamdeck-buttons/ledbutton"
	"github.com/dh1tw/touchctl/hub"
	rotatorpage "github.com/dh1tw/touchctl/pages/rotator"
)

type GenPage struct {
	sync.Mutex
	parent               esd.Page
	active               bool
	hub                  *hub.Hub
	sd                   *esd.StreamDeck
	staticLabels         map[int]*StaticLabelBtn
	rotators             map[int]*RotatorBtn
	terminalBtns         map[int]*TerminalBtn
	btnColorAvailable    color.Color
	btnColorNotAvailable color.Color
}

type StaticLabelBtn struct {
	*label.Label
}

type RotatorBtn struct {
	Name string
	*label.Label
	rotator.Rotator
}

type TerminalBtn struct {
	SwitchName   string
	TerminalName string
	TerminalText string
	*ledBtn.LedButton
	sw.Switcher
}

type Config struct {
	Layout []Btn
}

type BtnType int

const (
	Label BtnType = iota
	Back
	Rotator
	Band
	Terminal
)

type Btn struct {
	Type       BtnType
	DeviceName string
	ItemName   string
	Text       string
	Position   int
}

func NewGenPage(sd *esd.StreamDeck, h *hub.Hub, config Config) (*GenPage, error) {

	g := &GenPage{
		sd:                   sd,
		hub:                  h,
		staticLabels:         make(map[int]*StaticLabelBtn),
		rotators:             make(map[int]*RotatorBtn),
		terminalBtns:         make(map[int]*TerminalBtn),
		btnColorAvailable:    color.RGBA{255, 255, 255, 255}, //white
		btnColorNotAvailable: color.RGBA{80, 80, 80, 255},    //grey
	}

	// setup the layout
	for _, btn := range config.Layout {

		if err := g.checkDuplicateBtn(btn.Position); err != nil {
			return nil, err
		}

		switch btn.Type {
		case Label:
			x, err := label.NewLabel(sd, btn.Position, label.Text(btn.Text), label.TextColor(color.RGBA{92, 184, 92, 255}))
			if err != nil {
				return nil, err
			}
			g.staticLabels[btn.Position] = &StaticLabelBtn{
				Label: x}
		case Rotator:
			x, err := label.NewLabel(sd, btn.Position, label.Text("N/A"), label.TextColor(g.btnColorNotAvailable))
			if err != nil {
				return nil, err
			}
			g.rotators[btn.Position] = &RotatorBtn{
				Name:  btn.DeviceName,
				Label: x,
			}
		case Terminal:
			x, err := ledBtn.NewLedButton(sd, btn.Position, ledBtn.LedColor(ledBtn.LEDGreen), ledBtn.Text("N/A"), ledBtn.TextColor(g.btnColorNotAvailable), ledBtn.State(false))
			if err != nil {
				return nil, err
			}
			g.terminalBtns[btn.Position] = &TerminalBtn{
				TerminalName: btn.ItemName,
				TerminalText: btn.Text,
				SwitchName:   btn.DeviceName,
				LedButton:    x,
			}
		}
	}

	return g, nil
}

func (gp *GenPage) checkDuplicateBtn(pos int) error {

	dupe := false

	if _, duplicate := gp.staticLabels[pos]; duplicate {
		dupe = true
	}
	if _, duplicate := gp.rotators[pos]; duplicate {
		dupe = true
	}
	if _, duplicate := gp.terminalBtns[pos]; duplicate {
		dupe = true
	}

	if dupe {
		return fmt.Errorf("button %v already assigned", pos)
	}

	return nil
}

// HandleSbDeviceUpdate is called whenever the status of a Shackbus Device changes
func (gp *GenPage) HandleSbDeviceUpdate(event hub.SbDeviceStatusEvent) {
	gp.Lock()
	defer gp.Unlock()

	switch event.Event {
	case hub.SbAddDevice:
		gp.addSbDevice(event.DeviceName)
	case hub.SbRemoveDevice:
		gp.remoteSbDevice(event.DeviceName)
	case hub.SbUpdateDevice:
		gp.updateSbDevice(event.DeviceName)
	}
}

func (gp *GenPage) addSbDevice(newDeviceName string) {
	for _, r := range gp.rotators {
		if r.Name != newDeviceName {
			continue
		}
		newRotator, _ := gp.hub.Rotator(newDeviceName)
		r.Rotator = newRotator
		r.Label.SetTextColor(gp.btnColorAvailable)
		// r.Label.SetText(fmt.Sprintf("%03d°", r.Azimuth()))
		r.Label.SetText(fmt.Sprintf("%3v°", r.Azimuth()))
		if gp.active {
			r.Draw()
		}
	}

	for _, tBtn := range gp.terminalBtns {
		if tBtn.SwitchName != newDeviceName {
			continue
		}
		newStack, _ := gp.hub.Switch(newDeviceName)
		tBtn.Switcher = newStack
		port, err := tBtn.Switcher.GetPort("SM")
		if err != nil {
			log.Printf("unable to add stackmatch %v: port 'SM' not present", newDeviceName)
			continue
		}
		for _, t := range port.Terminals {
			if t.Name != tBtn.TerminalName {
				continue
			}
			tBtn.SetState(t.State)
			tBtn.SetTextColor(gp.btnColorAvailable)
			tBtn.SetText(tBtn.TerminalText)
			if gp.active {
				tBtn.Draw()
			}
		}
	}
}

func (gp *GenPage) updateSbDevice(newDeviceName string) {
	for _, r := range gp.rotators {
		if r.Name != newDeviceName {
			continue
		}
		if r.Rotator == nil {
			continue
		}
		// r.Label.SetText(fmt.Sprintf("%03d°", r.Azimuth()))
		r.Label.SetText(fmt.Sprintf("%3v°", r.Azimuth()))
		if gp.active {
			r.Draw()
		}
	}

	for _, tBtn := range gp.terminalBtns {
		if tBtn.SwitchName != newDeviceName {
			continue
		}
		if tBtn.Switcher == nil {
			continue
		}
		port, _ := tBtn.Switcher.GetPort("SM")
		for _, t := range port.Terminals {
			if t.Name != tBtn.TerminalName {
				continue
			}
			tBtn.SetState(t.State)
			if gp.active {
				tBtn.Draw()
			}
		}
	}
}

func (gp *GenPage) remoteSbDevice(remDeviceName string) {
	for _, r := range gp.rotators {
		if r.Name != remDeviceName {
			continue
		}
		r.Rotator = nil
		r.Label.SetTextColor(gp.btnColorNotAvailable)
		r.Label.SetText("N/A")
		if gp.active {
			r.Draw()
		}
	}

	for _, tBtn := range gp.terminalBtns {
		if tBtn.SwitchName != remDeviceName {
			continue
		}
		tBtn.Switcher = nil
		tBtn.SetState(false)
		tBtn.SetText("N/A")
		tBtn.SetTextColor(gp.btnColorNotAvailable)
		if gp.active {
			tBtn.Draw()
		}
	}
}

func (gp *GenPage) Set(btnIndex int, state esd.BtnState) esd.Page {
	gp.Lock()
	defer gp.Unlock()

	if state == esd.BtnReleased {
		return nil
	}

	r, exists := gp.rotators[btnIndex]
	if exists {
		if r.Rotator == nil {
			return nil
		}
		return rotatorpage.NewRotatorPage(gp.sd, gp, r.Rotator)
	}

	s, exists := gp.terminalBtns[btnIndex]
	if exists {
		if s.Switcher == nil {
			return nil
		}
		p := sw.Port{
			Name: "SM",
			Terminals: []sw.Terminal{
				sw.Terminal{
					Name:  s.TerminalName,
					State: !s.State(),
				},
			},
		}
		if err := s.Switcher.SetPort(p); err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (gp *GenPage) Parent() esd.Page {
	gp.Lock()
	defer gp.Unlock()
	return gp.parent
}

func (gp *GenPage) Draw() {
	gp.Lock()
	defer gp.Unlock()

	if !gp.active {
		return
	}

	for _, label := range gp.staticLabels {
		label.Draw()
	}

	for _, r := range gp.rotators {
		r.Draw()
	}

	for _, t := range gp.terminalBtns {
		t.Draw()
	}

}

func (gp *GenPage) SetActive(active bool) {
	gp.Lock()
	defer gp.Unlock()
	gp.active = active
}
