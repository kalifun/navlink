package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// startOrder stores the startup order of the components sorted by dependencies:
// Sample start-up order
//
//	startOrder = []string{
//		"EventBus", // Infrastructure, No Dependence
//		"SessionStore", // Depends on EventBus
//		"ComponentRegistry", // Depends on EventBus
//		"MessageRouter", // Depends on EventBus, Registry
//		"Transport", // Rely on EventBus, Router
//		"WebSocketTransport", // Depends on EventBus, Router
//	}
type LifecycleManager struct {
	components      []LifecycleComponent
	dependencies    map[string][]string // component -> dependencies
	startOrder      []string
	mu              sync.RWMutex
	started         bool
	stopped         bool
	shutdownTimeout time.Duration
}

type LifecycleComponent interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func (lm *LifecycleManager) Start(ctx context.Context) error {
    lm.mu.Lock()
    defer lm.mu.Unlock()

	if lm.started {
		return fmt.Errorf("lifecycle already started")
	}

	for _, componentID := range lm.startOrder {
		component := lm.getComponentByID(componentID)
		if component == nil {
			continue
		}

        // Start with timeout
        startCtx, cancel := context.WithTimeout(ctx, lm.shutdownTimeout)
        defer cancel()

        if err := component.Start(startCtx); err != nil {
            // Stop already started components
            lm.stopComponents(startCtx)
            return fmt.Errorf("failed to start component %s: %w", componentID, err)
        }
    }

	lm.started = true
	return nil
}

func (lm *LifecycleManager) Stop(ctx context.Context) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.stopped {
		return nil
	}

	lm.stopped = true
	return lm.stopComponents(ctx)
}

func (lm *LifecycleManager) AddComponent(component LifecycleComponent, dependencies ...string) error {
    lm.mu.Lock()
    defer lm.mu.Unlock()

    componentID := lm.getComponentID(component)

    if lm.dependencies == nil {
        lm.dependencies = make(map[string][]string)
    }

	if err := lm.checkCircularDependency(componentID, dependencies); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	lm.components = append(lm.components, component)
	lm.dependencies[componentID] = dependencies

    if err := lm.calculateStartOrder(); err != nil {
        return fmt.Errorf("failed to calculate start order: %w", err)
    }
    return nil
}

func (lm *LifecycleManager) getComponentID(component LifecycleComponent) string {
	return fmt.Sprintf("%T", component)
}

func (lm *LifecycleManager) checkCircularDependency(componentID string, dependencies []string) error {
	visited := make(map[string]bool)
	return lm.checkCircularDependencyRecursive(componentID, dependencies, visited)
}

func (lm *LifecycleManager) checkCircularDependencyRecursive(currentID string, dependencies []string, visited map[string]bool) error {
	for _, dep := range dependencies {
		if dep == currentID {
			return fmt.Errorf("circular dependency: %s depends on itself", currentID)
		}

		if visited[dep] {
			return fmt.Errorf("circular dependency detected involving %s", dep)
		}

		visited[dep] = true
		if deps, exist := lm.dependencies[dep]; exist {
			if err := lm.checkCircularDependencyRecursive(currentID, deps, visited); err != nil {
				return err
			}
		}

		delete(visited, dep)

	}
	return nil
}

func (lm *LifecycleManager) calculateStartOrder() error {
	graphOrder := make(map[string][]string)
	inDegree := make(map[string]int)

	for componentID, dependencies := range lm.dependencies {
		graphOrder[componentID] = dependencies
		inDegree[componentID] = 0
	}

	for _, deps := range lm.dependencies {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Find nodes with no dependencies
	queue := make([]string, 0)
	for componentID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, componentID)
		}
	}

	// Topological sort
	order := make([]string, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, neighbor := range graphOrder[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(order) != len(lm.dependencies) {
		return fmt.Errorf("circular dependency detected")
	}

	lm.startOrder = order
	return nil
}

func (lm *LifecycleManager) getComponentByID(id string) LifecycleComponent {
	for _, component := range lm.components {
		if lm.getComponentID(component) == id {
			return component
		}
	}

	return nil
}

func (lm *LifecycleManager) stopComponents(ctx context.Context) error {
	var lastErr error

	for i := len(lm.startOrder) - 1; i >= 0; i-- {
		componentID := lm.startOrder[i]
		component := lm.getComponentByID(componentID)
		if component == nil {
			continue
		}

		stopCtx, cancel := context.WithTimeout(ctx, lm.shutdownTimeout)
		defer cancel()

		if err := component.Stop(stopCtx); err != nil {
			lastErr = fmt.Errorf("failed to stop component %s: %w", componentID, err)
		}
	}

	return lastErr
}
