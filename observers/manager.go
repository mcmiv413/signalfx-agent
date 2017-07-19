package observers

import (
	"reflect"
	"sync"

	"github.com/signalfx/neo-agent/core/config"
	log "github.com/sirupsen/logrus"
)

type ObserverWrapper struct {
	instance interface{}
	_type    string
	// May be blank
	id         string
	lastConfig *config.ObserverConfig
	// Marked to be shutdown or not
	doomed bool
}

func (ow *ObserverWrapper) Shutdown() {
	if sh, ok := ow.instance.(Shutdownable); ok {
		sh.Shutdown()
	}
}

type ObserverManager struct {
	observers []*ObserverWrapper
	lock      sync.Mutex
	// Where to send observer notifications to
	CallbackTargets *ServiceCallbacks
}

func (om *ObserverManager) MakeWrappedObserver(config *config.ObserverConfig) *ObserverWrapper {
	factory, ok := observerFactories[config.Type]
	if !ok {
		log.WithFields(log.Fields{
			"observerType": config.Type,
			"id":           config.Id,
		}).Error("Observer type not recognized")
		return nil
	}

	if om.CallbackTargets == nil ||
		om.CallbackTargets.Added == nil ||
		om.CallbackTargets.Removed == nil {
		log.Fatal("om.CallbackTargets is not configured correctly, no point in observing")
	}

	return &ObserverWrapper{
		instance:   factory(om.CallbackTargets),
		lastConfig: config,
		_type:      config.Type,
		id:         config.Id,
		doomed:     false,
	}
}

// Configure will do everything it can and returns any errors it encounters
// with certain observers.  It is possible that some plugins might be
// configured and others not.
func (om *ObserverManager) Configure(obsConfig []config.ObserverConfig) {
	om.lock.Lock()
	defer om.lock.Unlock()

	if om.observers == nil {
		om.observers = make([]*ObserverWrapper, 0, len(obsConfig))
	}

	om.markAllAsDoomed()

OUTER:
	for i := range obsConfig {
		cfg := &obsConfig[i]

		// Find an existing observer of that same type that is marked for
		// shutdown and reuse it
		for _, obs := range om.observers {
			if obs._type == cfg.Type && obs.doomed {
				configEqual := reflect.DeepEqual(*obs.lastConfig, *cfg)

				if !configEqual {
					ok := configureObserver(obs.instance, cfg)
					if !ok {
						log.WithFields(log.Fields{
							"observerType": cfg.Type,
							"config":       cfg,
						}).Error("Could not configure observer")

						// Remains doomed if it misconfigures and isn't retried
						// successfully by another config of the same type
						continue OUTER
					}
				}
				obs.doomed = false
				continue OUTER
			}
		}

		// Couldn't reuse an existing observer so make a new one
		observer := om.MakeWrappedObserver(cfg)
		if observer == nil {
			continue
		}

		if !configureObserver(observer.instance, cfg) {
			continue
		}

		om.observers = append(om.observers, observer)
	}

	om.deleteDoomed()
}

func (om *ObserverManager) markAllAsDoomed() {
	for _, obs := range om.observers {
		obs.doomed = true
	}
}

func (om *ObserverManager) deleteDoomed() {
	// those saved from being doomed
	savedObservers := []*ObserverWrapper{}
	for _, ow := range om.observers {
		if ow.doomed {
			ow.Shutdown()
		} else {
			savedObservers = append(savedObservers, ow)
		}
	}
	om.observers = savedObservers
}

func (om *ObserverManager) Shutdown() {
	for i := range om.observers {
		om.observers[i].Shutdown()
	}
}
