package rotatorpage

import (
	"image/color"
	"log"
	"strconv"
	"sync"

	"github.com/dh1tw/remoteRotator/rotator"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck-buttons/label"
	presetpage "github.com/dh1tw/touchctl/pages/preset"
)

type rotatorPage struct {
	sync.Mutex
	sd            *esd.StreamDeck
	ownParent     esd.Page
	numPad        map[int]*label.Label
	newPos        *label.Label
	back          *label.Label
	set           *label.Label
	preset        *label.Label
	newPosText    string
	keyPadMapping map[int]int
	rotator       rotator.Rotator
	active        bool
}

func NewRotatorPage(sd *esd.StreamDeck, parent esd.Page, r rotator.Rotator) esd.Page {

	sp := &rotatorPage{
		sd:        sd,
		ownParent: parent,
		numPad:    make(map[int]*label.Label),
		keyPadMapping: map[int]int{
			10: 0,
			3:  1,
			2:  2,
			1:  3,
			8:  4,
			7:  5,
			6:  6,
			13: 7,
			12: 8,
			11: 9,
		},
		rotator: r,
	}

	newPos, err := label.NewLabel(sd, 0,
		label.BgColor(color.RGBA{0, 255, 0, 255}),
		label.TextColor(color.RGBA{0, 0, 0, 255}))
	if err != nil {
		log.Panic(err)
	}
	sp.newPos = newPos

	for pos, num := range sp.keyPadMapping {
		l, err := label.NewLabel(sd, pos,
			label.Text(strconv.Itoa(num)))
		// label.BgColor(color.RGBA{255, 255, 0, 0}),
		// label.TextColor(color.RGBA{0, 0, 0, 255}),

		if err != nil {
			log.Panic(err)
		}
		sp.numPad[pos] = l
	}

	set, err := label.NewLabel(sd, 5, label.Text("SET"))
	if err != nil {
		log.Panic(err)
	}
	sp.set = set

	ret, err := label.NewLabel(sd, 4, label.Text("BACK"))
	if err != nil {
		log.Panic(err)
	}
	sp.back = ret

	preset, err := label.NewLabel(sd, 9, label.Text("PSET"))
	if err != nil {
		log.Panic(err)
	}
	sp.preset = preset

	return sp
}

func (sp *rotatorPage) Set(btnIndex int, state esd.BtnState) esd.Page {
	sp.Lock()
	defer sp.Unlock()

	if state == esd.BtnPressed {
		return nil
	}

	switch btnIndex {
	case 4:
		return sp.parent()
	case 5:
		dir, err := strconv.Atoi(sp.newPosText)
		if err != nil {
			// log.Println(err)
			break
		}
		if dir < 0 || dir > 450 {
			sp.newPosText = ""
			sp.newPos.SetText(sp.newPosText)
			sp.newPos.Draw()
			break
		}
		sp.rotator.SetAzimuth(dir)
		return sp.parent()
	case 9:
		return presetpage.NewPresetPage(sp.sd, sp, sp.rotator)
	}

	_, ok := sp.numPad[btnIndex]
	if ok {
		if len(sp.newPosText) > 3 {
			return nil
		}
		num := sp.keyPadMapping[btnIndex]
		sp.newPosText = sp.newPosText + strconv.Itoa(num)
		sp.newPos.SetText(sp.newPosText)
		sp.draw()
	}

	return nil
}

func (sp *rotatorPage) draw() {
	for _, btn := range sp.numPad {
		btn.Draw()
	}
	sp.newPos.Draw()
	sp.preset.Draw()
	sp.back.Draw()
	sp.set.Draw()
}

func (sp *rotatorPage) Draw() {
	sp.Lock()
	defer sp.Unlock()
	sp.draw()
}

func (sp *rotatorPage) parent() esd.Page {
	return sp.ownParent
}

func (sp *rotatorPage) Parent() esd.Page {
	sp.Lock()
	defer sp.Unlock()
	return sp.parent()
}

func (sp *rotatorPage) SetActive(active bool) {
	sp.Lock()
	defer sp.Unlock()
	sp.active = active
}
