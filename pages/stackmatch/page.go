package stackpage

import (
	"fmt"
	"image/color"
	"log"
	"sort"
	"time"

	Switch "github.com/dh1tw/remoteSwitch/switch"

	"github.com/dh1tw/remoteRotator/rotator"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck/label"
	ledBtn "github.com/dh1tw/streamdeck/ledbutton"
	"github.com/dh1tw/touchctl/hub"
	rotatorpage "github.com/dh1tw/touchctl/pages/rotator"
)

type stackPage struct {
	sd       *esd.StreamDeck
	parent   esd.Page
	stack    stackmatch
	rotators map[int]*rot
	labels   map[int]*label.Label
	hub      *hub.Hub
	active   bool
}

type rot struct {
	rotator rotator.Rotator
	label   *label.Label
}

type stackmatch struct {
	sm   Switch.Switcher
	btns map[string]*ledBtn.LedButton
}

func (sm *stackmatch) update() error {
	p, err := sm.sm.GetPort("SM")
	if err != nil {
		return err
	}
	for _, t := range p.Terminals {
		btn, ok := sm.btns[t.Name]
		if !ok {
			log.Printf("unknown terminal %s", t.Name)
		}
		btn.SetState(t.State)
	}
	return nil
}

func (sm *stackmatch) set(terminalName string) error {
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

func NewStackPage(sd *esd.StreamDeck, parent esd.Page, h *hub.Hub) esd.Page {

	sp := &stackPage{
		sd:       sd,
		parent:   parent,
		rotators: make(map[int]*rot, 0),
		labels:   make(map[int]*label.Label),
		hub:      h,
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

	bandLabel, err := label.NewLabel(sd, 14, label.Text("20m"), label.TextColor(color.RGBA{255, 0, 0, 255}))
	if err != nil {
		log.Fatal(err)
	}

	sp.labels[14] = bandLabel

	rots := sp.hub.Rotators()
	sort.Slice(rots, func(i, j int) bool {
		return rots[i].Name() < rots[j].Name()
	})

	stack20m, exists := sp.hub.Switch("Stackmatch 20m")
	if !exists {
		log.Fatalf("Stackmatch 20m doesn't exist")
	}

	port, err := stack20m.GetPort("SM")
	if err != nil {
		log.Fatalf("port SM doesn't exist")
	}

	sm := stackmatch{
		sm:   stack20m,
		btns: make(map[string]*ledBtn.LedButton),
	}

	for _, t := range port.Terminals {

		switch t.Name {
		case "OB11-TWR1":
			b, err := ledBtn.NewLedButton(sd, 13, ledBtn.Text("OB11"))
			if err != nil {
				log.Fatal(err)
			}
			b.SetState(t.State)
			sm.btns["OB11-TWR1"] = b
		case "4L-TWR2":
			b, err := ledBtn.NewLedButton(sd, 12, ledBtn.Text(" 4L "))
			if err != nil {
				log.Fatal(err)
			}
			b.SetState(t.State)
			sm.btns["4L-TWR2"] = b
		case "OB11-TWR3":
			b, err := ledBtn.NewLedButton(sd, 11, ledBtn.Text("OB11"))
			if err != nil {
				log.Fatal(err)
			}
			b.SetState(t.State)
			sm.btns["OB11-TWR3"] = b
		}
	}

	sp.stack = sm

	counter := 8
	for _, r := range rots {
		lbl, err := label.NewLabel(sd, counter, label.Text(fmt.Sprintf("03%d°", r.Azimuth())))
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

	go func() {
		ticker := time.NewTicker(time.Millisecond * 200)
		for {
			select {
			case <-ticker.C:
				for _, r := range sp.rotators {
					r.label.SetText(fmt.Sprintf("%03d°", r.rotator.Azimuth()))
				}
				sp.stack.update()

				if sp.active {
					sp.Draw()
				}
			}
		}
	}()

	sp.SetActive(true)

	return sp
}

func (sp *stackPage) Set(btnIndex int, state esd.BtnState) esd.Page {

	if state == esd.BtnReleased {
		return nil
	}

	switch btnIndex {
	case 11:
		if err := sp.stack.set("OB11-TWR3"); err != nil {
			log.Println(err)
		}
	case 12:
		if err := sp.stack.set("4L-TWR2"); err != nil {
			log.Println(err)
		}
	case 13:
		if err := sp.stack.set("OB11-TWR1"); err != nil {
			log.Println(err)
		}
	default: // rotator
		rot, ok := sp.rotators[btnIndex]
		if ok {
			sp.SetActive(false)
			return rotatorpage.NewRotatorPage(sp.sd, sp, rot.rotator)
		}
	}

	return nil
}

func (sp *stackPage) SetActive(active bool) {
	sp.active = active
}

func (sp *stackPage) Draw() {
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

func (sp *stackPage) Parent() esd.Page {
	return sp.parent
}
