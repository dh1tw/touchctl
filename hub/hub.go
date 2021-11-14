package hub

import (
	"fmt"
	"log"
	"sync"

	"github.com/dh1tw/remoteRotator/rotator"
	Switch "github.com/dh1tw/remoteSwitch/switch"
)

// Hub is a struct which makes a rotator available through network
// interfaces, supporting several protocols.
type Hub struct {
	sync.RWMutex
	rotators                  map[string]rotator.Rotator //key: Rotator name
	switches                  map[string]Switch.Switcher //key: Switch name
	sbDeviceStatusSubscribers map[string]SbDeviceStatusCb
}

type SbDeviceStatusCb func(ev SbDeviceStatusEvent)

// NewHub returns the pointer to an initialized Hub object.
func NewHub(rotators ...rotator.Rotator) (*Hub, error) {
	hub := &Hub{
		rotators:                  make(map[string]rotator.Rotator),
		switches:                  make(map[string]Switch.Switcher),
		sbDeviceStatusSubscribers: make(map[string]SbDeviceStatusCb),
	}

	for _, r := range rotators {
		if err := hub.AddRotator(r); err != nil {
			return nil, err
		}
	}

	go hub.handleClose()

	return hub, nil
}

func (hub *Hub) handleClose() {
}

// AddRotator adds / registers a rotator. The rotator's name must be unique.
func (hub *Hub) AddRotator(r rotator.Rotator) error {
	hub.Lock()
	defer hub.Unlock()

	return hub.addRotator(r)
}

// AddSwitch adds / registers a rotator. The rotator's name must be unique.
func (hub *Hub) AddSwitch(s Switch.Switcher) error {
	hub.Lock()
	defer hub.Unlock()

	return hub.addSwitch(s)
}

func (hub *Hub) addRotator(r rotator.Rotator) error {
	_, ok := hub.rotators[r.Name()]
	if ok {
		return fmt.Errorf("rotator names must be unique; %s provided twice", r.Name())
	}
	hub.rotators[r.Name()] = r

	ev := SbDeviceStatusEvent{
		Event:      SbAddDevice,
		DeviceName: r.Name(),
	}
	if err := hub.broadcastSbDeviceStatus(ev); err != nil {
		log.Println(err)
	}

	log.Printf("added rotator (%s)\n", r.Name())

	return nil
}

func (hub *Hub) addSwitch(s Switch.Switcher) error {
	_, ok := hub.switches[s.Name()]
	if ok {
		return fmt.Errorf("the switch's names must be unique; %s provided twice", s.Name())
	}
	hub.switches[s.Name()] = s

	ev := SbDeviceStatusEvent{
		Event:      SbAddDevice,
		DeviceName: s.Name(),
	}
	if err := hub.broadcastSbDeviceStatus(ev); err != nil {
		log.Println(err)
	}

	log.Printf("added switch (%s)\n", s.Name())

	return nil
}

// RemoveRotator deletes / de-registers a rotator.
func (hub *Hub) RemoveRotator(r rotator.Rotator) {
	hub.Lock()
	defer hub.Unlock()

	r.Close()
	delete(hub.rotators, r.Name())

	ev := SbDeviceStatusEvent{
		Event:      SbRemoveDevice,
		DeviceName: r.Name(),
	}
	if err := hub.broadcastSbDeviceStatus(ev); err != nil {
		log.Println(err)
	}
	log.Printf("removed rotator (%s)\n", r.Name())
}

// RemoveSwitch deletes / de-registers a switch.
func (hub *Hub) RemoveSwitch(s Switch.Switcher) {
	hub.Lock()
	defer hub.Unlock()

	s.Close()
	delete(hub.switches, s.Name())

	ev := SbDeviceStatusEvent{
		Event:      SbRemoveDevice,
		DeviceName: s.Name(),
	}
	if err := hub.broadcastSbDeviceStatus(ev); err != nil {
		log.Println(err)
	}

	log.Printf("removed switch (%s)\n", s.Name())
}

// Rotator returns a particular rotator stored from the hub. If no
// rotator exists with that name, (nil, false) will be returned.
func (hub *Hub) Rotator(name string) (rotator.Rotator, bool) {
	hub.RLock()
	defer hub.RUnlock()

	rotator, ok := hub.rotators[name]
	return rotator, ok
}

// Switch returns a particular switch stored from the hub. If no
// switch exists with that name, (nil, false) will be returned.
func (hub *Hub) Switch(name string) (Switch.Switcher, bool) {
	hub.RLock()
	defer hub.RUnlock()

	sw, ok := hub.switches[name]
	return sw, ok
}

// Rotators returns a slice of all registered rotators.
func (hub *Hub) Rotators() []rotator.Rotator {
	hub.RLock()
	defer hub.RUnlock()

	rotators := make([]rotator.Rotator, 0, len(hub.rotators))
	for _, r := range hub.rotators {
		rotators = append(rotators, r)
	}

	return rotators
}

// Switches returns a slice of all registered switches.
func (hub *Hub) Switches() []Switch.Switcher {
	hub.RLock()
	defer hub.RUnlock()

	switches := make([]Switch.Switcher, 0, len(hub.switches))
	for _, r := range hub.switches {
		switches = append(switches, r)
	}

	return switches
}

type SbDeviceStatusEvent struct {
	Event      SbDeviceStatusEventType `json:"event,omitempty"`
	DeviceName string                  `json:"device_name,omitempty"`
}

type SbDeviceStatusEventType string

const (
	SbAddDevice    SbDeviceStatusEventType = "add"
	SbRemoveDevice SbDeviceStatusEventType = "remove"
	SbUpdateDevice SbDeviceStatusEventType = "update"
)

func (hub *Hub) SubscribeToSbDeviceStatus(subscriberName string, cb SbDeviceStatusCb) error {
	hub.Lock()
	defer hub.Unlock()

	if _, exists := hub.sbDeviceStatusSubscribers[subscriberName]; exists {
		return fmt.Errorf("subscriber '%v' already exists", subscriberName)
	}

	hub.sbDeviceStatusSubscribers[subscriberName] = cb

	return nil
}

func (hub *Hub) UnsubscribeFromSbDeviceStatus(subscriberName string) {
	hub.Lock()
	defer hub.Unlock()

	delete(hub.sbDeviceStatusSubscribers, subscriberName)
}

func (hub *Hub) BroadcastSbDeviceStatus(event SbDeviceStatusEvent) error {
	hub.Lock()
	defer hub.Unlock()

	return hub.broadcastSbDeviceStatus(event)
}

func (hub *Hub) broadcastSbDeviceStatus(event SbDeviceStatusEvent) error {

	for sub, cb := range hub.sbDeviceStatusSubscribers {

		if cb == nil {
			delete(hub.sbDeviceStatusSubscribers, sub)
			log.Printf("removed orphaned sbDeviceStatus Subscriber '%v'\n", sub)
		}

		go cb(event)
	}

	return nil
}
