package bandpage

import (
	"image/color"
	"log"
	"sync"

	esd "github.com/dh1tw/streamdeck"
	"github.com/dh1tw/streamdeck-buttons/label"
)

type bandPage struct {
	sync.Mutex
	sd        *esd.StreamDeck
	ownParent esd.Page
	active    bool
	labels    map[int]*bandButton
	stacks    map[string]esd.Page
}

type bandButton struct {
	name      string
	shortName string
	label     *label.Label
}

func NewBandPage(sd *esd.StreamDeck, parent esd.Page, stacks map[string]esd.Page) esd.Page {

	bp := &bandPage{
		sd:        sd,
		ownParent: parent,
		stacks:    stacks,
		labels: map[int]*bandButton{
			8:  &bandButton{name: "6m", shortName: " 6m "},
			7:  &bandButton{name: "10m", shortName: " 10m"},
			6:  &bandButton{name: "15m", shortName: " 15m"},
			5:  &bandButton{name: "20m", shortName: " 20m"},
			12: &bandButton{name: "40m", shortName: " 40m"},
			11: &bandButton{name: "80m", shortName: " 80m"},
			10: &bandButton{name: "160m", shortName: "160m"},
		},
	}

	for pos, l := range bp.labels {
		tb, err := label.NewLabel(sd, pos, label.Text(l.shortName), label.TextColor(color.RGBA{255, 0, 0, 255}))
		if err != nil {
			log.Fatal(err)
		}
		bp.labels[pos].label = tb
	}

	return bp
}

func (bp *bandPage) Set(btnIndex int, state esd.BtnState) esd.Page {

	bp.Lock()
	defer bp.Unlock()
	if state == esd.BtnReleased {
		return nil
	}

	if bandBtn, ok := bp.labels[btnIndex]; ok {
		if stack, ok := bp.stacks[bandBtn.name]; ok {
			return stack
		}
	}

	return nil
}

func (bp *bandPage) SetActive(active bool) {
	bp.Lock()
	defer bp.Unlock()
	bp.active = active
}

func (bp *bandPage) draw() {
	for _, label := range bp.labels {
		label.label.Draw()
	}
}

func (bp *bandPage) Draw() {
	bp.Lock()
	defer bp.Unlock()
	bp.draw()
}

func (bp *bandPage) parent() esd.Page {
	return bp.ownParent
}

func (bp *bandPage) Parent() esd.Page {
	return bp.parent()
}
