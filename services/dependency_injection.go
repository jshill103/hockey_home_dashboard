package services

import (
	"fmt"
	"reflect"
	"sync"
)

// ServiceContainer manages dependency injection
type ServiceContainer struct {
	services map[string]interface{}
	mu       sync.RWMutex

	// Lifecycle hooks
	initCallbacks     []func() error
	shutdownCallbacks []func() error
}

var (
	globalContainer     *ServiceContainer
	globalContainerOnce sync.Once
)

// InitServiceContainer initializes the global service container
func InitServiceContainer() *ServiceContainer {
	globalContainerOnce.Do(func() {
		globalContainer = &ServiceContainer{
			services:          make(map[string]interface{}),
			initCallbacks:     make([]func() error, 0),
			shutdownCallbacks: make([]func() error, 0),
		}
		fmt.Println("✅ Service Container initialized")
	})
	return globalContainer
}

// GetServiceContainer returns the global service container
func GetServiceContainer() *ServiceContainer {
	return globalContainer
}

// Register adds a service to the container
func (sc *ServiceContainer) Register(name string, service interface{}) error {
	if sc == nil {
		return fmt.Errorf("service container not initialized")
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	if _, exists := sc.services[name]; exists {
		return fmt.Errorf("service '%s' already registered", name)
	}

	sc.services[name] = service
	return nil
}

// MustRegister registers a service and panics on error
func (sc *ServiceContainer) MustRegister(name string, service interface{}) {
	if err := sc.Register(name, service); err != nil {
		panic(err)
	}
}

// Get retrieves a service from the container
func (sc *ServiceContainer) Get(name string) (interface{}, error) {
	if sc == nil {
		return nil, fmt.Errorf("service container not initialized")
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	service, exists := sc.services[name]
	if !exists {
		return nil, fmt.Errorf("service '%s' not found", name)
	}

	return service, nil
}

// MustGet retrieves a service and panics if not found
func (sc *ServiceContainer) MustGet(name string) interface{} {
	service, err := sc.Get(name)
	if err != nil {
		panic(err)
	}
	return service
}

// GetTyped retrieves a service with type assertion
func GetTyped[T any](sc *ServiceContainer, name string) (T, error) {
	var zero T

	service, err := sc.Get(name)
	if err != nil {
		return zero, err
	}

	typed, ok := service.(T)
	if !ok {
		return zero, fmt.Errorf("service '%s' is not of type %T", name, zero)
	}

	return typed, nil
}

// MustGetTyped retrieves a service with type assertion and panics on error
func MustGetTyped[T any](sc *ServiceContainer, name string) T {
	service, err := GetTyped[T](sc, name)
	if err != nil {
		panic(err)
	}
	return service
}

// Has checks if a service is registered
func (sc *ServiceContainer) Has(name string) bool {
	if sc == nil {
		return false
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	_, exists := sc.services[name]
	return exists
}

// Remove removes a service from the container
func (sc *ServiceContainer) Remove(name string) {
	if sc == nil {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	delete(sc.services, name)
}

// List returns all registered service names
func (sc *ServiceContainer) List() []string {
	if sc == nil {
		return nil
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	names := make([]string, 0, len(sc.services))
	for name := range sc.services {
		names = append(names, name)
	}

	return names
}

// Count returns the number of registered services
func (sc *ServiceContainer) Count() int {
	if sc == nil {
		return 0
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return len(sc.services)
}

// OnInit registers a callback to run during initialization
func (sc *ServiceContainer) OnInit(callback func() error) {
	if sc == nil {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.initCallbacks = append(sc.initCallbacks, callback)
}

// OnShutdown registers a callback to run during shutdown
func (sc *ServiceContainer) OnShutdown(callback func() error) {
	if sc == nil {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.shutdownCallbacks = append(sc.shutdownCallbacks, callback)
}

// RunInitCallbacks executes all initialization callbacks
func (sc *ServiceContainer) RunInitCallbacks() error {
	if sc == nil {
		return nil
	}

	sc.mu.RLock()
	callbacks := make([]func() error, len(sc.initCallbacks))
	copy(callbacks, sc.initCallbacks)
	sc.mu.RUnlock()

	for i, callback := range callbacks {
		if err := callback(); err != nil {
			return fmt.Errorf("init callback %d failed: %w", i, err)
		}
	}

	return nil
}

// RunShutdownCallbacks executes all shutdown callbacks
func (sc *ServiceContainer) RunShutdownCallbacks() error {
	if sc == nil {
		return nil
	}

	sc.mu.RLock()
	callbacks := make([]func() error, len(sc.shutdownCallbacks))
	copy(callbacks, sc.shutdownCallbacks)
	sc.mu.RUnlock()

	// Run shutdown callbacks in reverse order
	var errors []error
	for i := len(callbacks) - 1; i >= 0; i-- {
		if err := callbacks[i](); err != nil {
			errors = append(errors, fmt.Errorf("shutdown callback %d failed: %w", i, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// GetServiceInfo returns information about a registered service
func (sc *ServiceContainer) GetServiceInfo(name string) map[string]interface{} {
	if sc == nil {
		return nil
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	service, exists := sc.services[name]
	if !exists {
		return nil
	}

	t := reflect.TypeOf(service)
	info := map[string]interface{}{
		"name":   name,
		"type":   t.String(),
		"kind":   t.Kind().String(),
		"is_nil": service == nil,
	}

	// Check if service has common methods
	v := reflect.ValueOf(service)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		methods := []string{}
		for i := 0; i < t.NumMethod(); i++ {
			methods = append(methods, t.Method(i).Name)
		}
		info["methods"] = methods
	}

	return info
}

// GetAllServiceInfo returns information about all registered services
func (sc *ServiceContainer) GetAllServiceInfo() map[string]interface{} {
	if sc == nil {
		return nil
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	allInfo := make(map[string]interface{})
	for name := range sc.services {
		allInfo[name] = sc.GetServiceInfo(name)
	}

	return allInfo
}

// Convenience functions for global container

// RegisterGlobal registers a service in the global container
func RegisterGlobal(name string, service interface{}) error {
	return GetServiceContainer().Register(name, service)
}

// MustRegisterGlobal registers a service in the global container and panics on error
func MustRegisterGlobal(name string, service interface{}) {
	GetServiceContainer().MustRegister(name, service)
}

// GetGlobal retrieves a service from the global container
func GetGlobal(name string) (interface{}, error) {
	return GetServiceContainer().Get(name)
}

// MustGetGlobal retrieves a service from the global container and panics if not found
func MustGetGlobal(name string) interface{} {
	return GetServiceContainer().MustGet(name)
}

// GetGlobalTyped retrieves a typed service from the global container
func GetGlobalTyped[T any](name string) (T, error) {
	return GetTyped[T](GetServiceContainer(), name)
}

// MustGetGlobalTyped retrieves a typed service from the global container and panics on error
func MustGetGlobalTyped[T any](name string) T {
	return MustGetTyped[T](GetServiceContainer(), name)
}

// HasGlobal checks if a service is registered in the global container
func HasGlobal(name string) bool {
	return GetServiceContainer().Has(name)
}
