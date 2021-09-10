package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/client"
	sbRotatorProxy "github.com/dh1tw/remoteRotator/rotator/sb_proxy"
	sbSwitchProxy "github.com/dh1tw/remoteSwitch/switch/sbSwitchProxy"
	"github.com/dh1tw/touchctl/hub"
)

type serviceCache struct {
	sync.Mutex
	ttl   time.Duration
	cache map[string]time.Time
}

type webserver struct {
	*hub.Hub
	cli   client.Client
	cache *serviceCache
}

// isRotator checks a serviceName string if it is a shackbus rotator
func isRotator(serviceName string) bool {

	if !strings.Contains(serviceName, "shackbus.rotator.") {
		return false
	}
	return true
}

// isSwitch checks a serviceName string if it is a shackbus switch
func isSwitch(serviceName string) bool {

	if !strings.Contains(serviceName, "shackbus.switch.") {
		return false
	}
	return true
}

// watchRegistry is a blocking function which continously
// checks the registry for changes (new rotators / switches being added/updated/removed).
func (w *webserver) watchRegistry() {
	watcher, err := w.cli.Options().Registry.Watch()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	for {
		res, err := watcher.Next()
		objType := ""

		if err != nil {
			log.Println("watch error:", err)
		}

		_isRotator := isRotator(res.Service.Name)
		_isSwitch := isSwitch(res.Service.Name)
		if !_isRotator && !_isSwitch {
			continue
		}

		if _isRotator {
			objType = "rotator"
		}

		if _isSwitch {
			objType = "switch"
		}

		switch res.Action {

		case "create", "update":
			switch objType {
			case "rotator":
				if err := w.addRotator(res.Service.Name); err != nil {
					log.Println(err)
				}
			case "switch":
				if err := w.addSwitch(res.Service.Name); err != nil {
					log.Println(err)
				}
			default:
				continue
			}
			w.cache.Lock()
			w.cache.cache[res.Service.Name] = time.Now()
			w.cache.Unlock()

		case "delete":
			serviceName := nameFromFQSN(res.Service.Name)

			switch objType {
			case "rotator":
				r, exists := w.Rotator(serviceName)
				if !exists {
					continue
				}
				r.Close()
			case "switch":
				s, exists := w.Switch(serviceName)
				if !exists {
					continue
				}
				s.Close()
			}

			w.cache.Lock()
			delete(w.cache.cache, res.Service.Name)
			w.cache.Unlock()
		}

		w.cache.Lock()
		for service, timeout := range w.cache.cache {
			if time.Since(timeout) >= w.cache.ttl {
				serviceName := nameFromFQSN(service)
				_isRotator := isRotator(serviceName)
				_isSwitch := isSwitch(serviceName)

				if _isRotator {
					r, exists := w.Rotator(serviceName)
					if !exists {
						continue
					}
					r.Close()
					delete(w.cache.cache, res.Service.Name)
				} else if _isSwitch {
					s, exists := w.Switch(serviceName)
					if !exists {
						continue
					}
					s.Close()
					delete(w.cache.cache, res.Service.Name)
				}
			}
		}
		w.cache.Unlock()
	}
}

//extract the service's name from its fully qualified service name (FQSN)
func nameFromFQSN(serviceName string) string {
	splitted := strings.Split(serviceName, ".")
	name := splitted[len(splitted)-1]
	return strings.Replace(name, "_", " ", -1)
}

func (w *webserver) addRotator(rotatorServiceName string) error {

	rotatorName := nameFromFQSN(rotatorServiceName)

	// only continue if this rotator(name) does not exist yet
	_, exists := w.Rotator(rotatorName)
	if exists {
		return nil
	}

	doneCh := make(chan struct{})

	done := sbRotatorProxy.DoneCh(doneCh)
	cli := sbRotatorProxy.Client(w.cli)
	eh := sbRotatorProxy.EventHandler(ev)
	name := sbRotatorProxy.Name(rotatorName)
	serviceName := sbRotatorProxy.ServiceName(strings.Replace(rotatorServiceName, " ", "_", -1))

	// create new rotator proxy object
	r, err := sbRotatorProxy.New(done, cli, eh, name, serviceName)
	if err != nil {
		close(doneCh)
		return fmt.Errorf("unable to create proxy object: %v", err)
	}

	if err := w.AddRotator(r); err != nil {
		close(doneCh)
		return fmt.Errorf("unable to add proxy objects: %v", err)
	}

	go func() {
		<-doneCh
		fmt.Println("disposing:", r.Name())
		w.RemoveRotator(r)
	}()

	return nil
}

func (w *webserver) addSwitch(switchServiceName string) error {

	switchName := nameFromFQSN(switchServiceName)

	// only continue if this rotator(name) does not exist yet
	_, exists := w.Switch(switchName)
	if exists {
		return nil
	}

	doneCh := make(chan struct{})

	done := sbSwitchProxy.DoneCh(doneCh)
	cli := sbSwitchProxy.Client(w.cli)
	// eh := sbSwitchProxy.EventHandler(ev)
	name := sbSwitchProxy.Name(switchName)
	serviceName := sbSwitchProxy.ServiceName(strings.Replace(switchServiceName, " ", "_", -1))

	// create new switch proxy object
	// s, err := sbSwitchProxy.New(done, cli, eh, name, serviceName)
	s, err := sbSwitchProxy.New(done, cli, name, serviceName)
	if err != nil {
		close(doneCh)
		return fmt.Errorf("unable to create proxy object: %v", err)
	}

	if err := w.AddSwitch(s); err != nil {
		close(doneCh)
		return fmt.Errorf("unable to add proxy objects: %v", err)
	}

	go func() {
		<-doneCh
		fmt.Println("disposing:", s.Name())
		w.RemoveSwitch(s)
	}()

	return nil
}

// listAndAddServices is a convenience function which queries the
// registry for all services and then add proxy objects for
// each of them.
func (w *webserver) listAndAddServices() error {

	services, err := w.cli.Options().Registry.ListServices()
	if err != nil {
		return err
	}

	for _, service := range services {
		fmt.Println("found:", service.Name)
		_isRotator := isRotator(service.Name)
		_isSwitch := isSwitch(service.Name)

		if !_isRotator && !_isSwitch {
			continue
		}

		if _isRotator {
			if err := w.addRotator(service.Name); err != nil {
				log.Println(err)
			}
		} else if _isSwitch {
			if err := w.addSwitch(service.Name); err != nil {
				log.Println(err)
			}

		}
	}

	return nil
}
