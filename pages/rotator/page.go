package rotatorpage

import (
	"image/color"
	"log"
	"strconv"

	"github.com/dh1tw/remoteRotator/rotator"
	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck/label"
	presetpage "github.com/dh1tw/touchctl/pages/preset"
)

type rotatorPage struct {
	sd            *esd.StreamDeck
	parent        esd.Page
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
		sd:     sd,
		parent: parent,
		numPad: make(map[int]*label.Label),
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

	sd.ClearAllBtns()

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
	if state == esd.BtnReleased {
		return nil
	}

	switch btnIndex {
	case 4:
		sp.parent.SetActive(true)
		return sp.parent
	case 5:
		dir, err := strconv.Atoi(sp.newPosText)
		if err != nil {
			log.Println(err)
			break
		}
		sp.rotator.SetAzimuth(dir)
		sp.parent.SetActive(true)
		return sp.parent
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
		sp.Draw()
	}

	return nil
}

func (sp *rotatorPage) Draw() {
	for _, btn := range sp.numPad {
		btn.Draw()
	}
	sp.newPos.Draw()
	sp.preset.Draw()
	sp.back.Draw()
	sp.set.Draw()
}

func (sp *rotatorPage) Parent() esd.Page {
	return sp.parent
}

func (sp *rotatorPage) SetActive(active bool) {
	sp.active = active
}
