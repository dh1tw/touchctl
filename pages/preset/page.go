package presetpage

import (
	"log"
	"sync"

	"github.com/dh1tw/remoteRotator/rotator"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck-buttons/label"
)

type presetPage struct {
	sync.Mutex
	sd         *esd.StreamDeck
	ownParent  esd.Page
	btns       map[int]*label.Label
	btnMapping map[int]presetValue
	back       *label.Label
	active     bool
	rotator    rotator.Rotator
}

type presetValue struct {
	text  string
	value int
}

func NewPresetPage(sd *esd.StreamDeck, parent esd.Page, r rotator.Rotator) esd.Page {

	pp := &presetPage{
		sd:        sd,
		ownParent: parent,
		rotator:   r,
		btns:      make(map[int]*label.Label),
		btnMapping: map[int]presetValue{
			3:  presetValue{"NW", 315},
			2:  presetValue{"N", 0},
			1:  presetValue{"NE", 45},
			8:  presetValue{"W", 270},
			6:  presetValue{"E", 90},
			13: presetValue{"SW", 225},
			12: presetValue{"S", 180},
			11: presetValue{"SE", 135},
			0:  presetValue{"NA", 320},
			5:  presetValue{"KH6", 350},
			10: presetValue{"VK", 75},
		},
	}

	for pos, v := range pp.btnMapping {
		l, err := label.NewLabel(sd, pos, label.Text(v.text))
		if err != nil {
			log.Panic(err)
		}
		pp.btns[pos] = l
	}

	back, err := label.NewLabel(sd, 4, label.Text("BACK"))
	if err != nil {
		log.Panic(err)
	}
	pp.back = back

	return pp
}

func (pp *presetPage) Set(btnIndex int, state esd.BtnState) esd.Page {
	pp.Lock()
	defer pp.Unlock()

	if state == esd.BtnReleased {
		return nil
	}

	switch btnIndex {
	case 4:
		return pp.parent()
	}

	v, ok := pp.btnMapping[btnIndex]
	if !ok {
		return nil
	}

	err := pp.rotator.SetAzimuth(v.value)
	if err != nil {
		log.Println(err)
	}

	pp.parent().Parent().SetActive(true)
	return pp.parent().Parent()
}

func (pp *presetPage) draw() {
	for _, btn := range pp.btns {
		btn.Draw()
	}
	pp.back.Draw()
}

func (pp *presetPage) Draw() {
	pp.Lock()
	defer pp.Unlock()
	pp.draw()
}

func (pp *presetPage) parent() esd.Page {
	return pp.ownParent
}

func (pp *presetPage) Parent() esd.Page {
	pp.Lock()
	defer pp.Unlock()
	return pp.parent()
}

func (pp *presetPage) setActive(active bool) {
	pp.active = active
}

func (pp *presetPage) SetActive(active bool) {
	pp.Lock()
	defer pp.Unlock()
	pp.setActive(active)
}
