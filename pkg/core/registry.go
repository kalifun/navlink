package core

import (
	"fmt"
	"sync"

	"github.com/kalifun/navlink/pkg/converter"
)

// ComponentRegistry manages component registration and discovery
type ComponentRegistry struct {
	transports map[string]TransportFactory
	converters map[string]converter.ConverterFactory
	processors map[string]ProcessorFactory
	mu         sync.RWMutex
}

func (cr *ComponentRegistry) RegisterTransport(name string, factory TransportFactory) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if _, exists := cr.transports[name]; exists {
		return fmt.Errorf("transport already registered: %s", name)
	}
	cr.transports[name] = factory
	return nil
}

func (cr *ComponentRegistry) RegisterProcessor(name string, factory ProcessorFactory) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if _, exists := cr.processors[name]; exists {
		return fmt.Errorf("processor already registered: %s", name)
	}
	cr.processors[name] = factory
	return nil
}

func (cr *ComponentRegistry) RegisterConverter(name string, factory converter.ConverterFactory) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if _, exists := cr.converters[name]; exists {
		return fmt.Errorf("converter already registered: %s", name)
	}
	cr.converters[name] = factory
	return nil
}

func (cr *ComponentRegistry) GetTransport(name string) (TransportFactory, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	factory, exists := cr.transports[name]
	if !exists {
		return nil, fmt.Errorf("transport not found: %s", name)
	}
	return factory, nil
}

func (cr *ComponentRegistry) GetProcessor(name string) (ProcessorFactory, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	factory, exists := cr.processors[name]
	if !exists {
		return nil, fmt.Errorf("processor not found: %s", name)
	}
	return factory, nil
}

func (cr *ComponentRegistry) GetConverter(name string) (converter.ConverterFactory, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	factory, exists := cr.converters[name]
	if !exists {
		return nil, fmt.Errorf("converter not found: %s", name)
	}
	return factory, nil
}

func (cr *ComponentRegistry) ListTransports() []string {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	names := make([]string, 0, len(cr.transports))
	for name := range cr.transports {
		names = append(names, name)
	}
	return names
}

func (cr *ComponentRegistry) ListProcessors() []string {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	names := make([]string, 0, len(cr.processors))
	for name := range cr.processors {
		names = append(names, name)
	}
	return names
}

func (cr *ComponentRegistry) ListConverters() []string {
    cr.mu.RLock()
    defer cr.mu.RUnlock()

	names := make([]string, 0, len(cr.converters))
	for name := range cr.converters {
		names = append(names, name)
	}
	return names
}
