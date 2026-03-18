// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"fmt"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
)

// ProviderFactoryFunc creates a DataProvider from a ProviderConfig.
// Each provider type (tugboat, accountablehq, gdrive) registers a factory.
type ProviderFactoryFunc func(pc config.ProviderConfig, log logger.Logger) (interfaces.DataProvider, error)

// factories maps provider type names to their factory functions.
var factories = map[string]ProviderFactoryFunc{}

// RegisterFactory registers a factory function for a provider type.
// Call this from provider packages' init() or from explicit setup code.
func RegisterFactory(typeName string, factory ProviderFactoryFunc) {
	factories[typeName] = factory
}

// RegisteredFactoryTypes returns the names of all registered factory types.
func RegisteredFactoryTypes() []string {
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}

// InitFromConfig reads the providers section of the config and registers
// enabled providers in the given registry. Providers whose type has no
// registered factory are skipped with a warning.
//
// Returns the number of providers registered and any errors encountered.
func InitFromConfig(cfg config.ProvidersConfig, registry *ProviderRegistry, log logger.Logger) (int, []error) {
	var errs []error
	registered := 0

	for _, pc := range cfg.Providers {
		if !pc.Enabled {
			log.Debug("Skipping disabled provider",
				logger.Field{Key: "name", Value: pc.Name},
				logger.Field{Key: "type", Value: pc.Type})
			continue
		}

		factory, ok := factories[pc.Type]
		if !ok {
			err := fmt.Errorf("no factory registered for provider type %q (provider %q)", pc.Type, pc.Name)
			log.Warn("Skipping unknown provider type", logger.Error(err))
			errs = append(errs, err)
			continue
		}

		provider, err := factory(pc, log)
		if err != nil {
			err = fmt.Errorf("failed to create provider %q (type %q): %w", pc.Name, pc.Type, err)
			log.Error("Provider creation failed", logger.Error(err))
			errs = append(errs, err)
			continue
		}

		if err := registry.Register(provider); err != nil {
			err = fmt.Errorf("failed to register provider %q: %w", pc.Name, err)
			log.Error("Provider registration failed", logger.Error(err))
			errs = append(errs, err)
			continue
		}

		log.Info("Registered provider from config",
			logger.Field{Key: "name", Value: pc.Name},
			logger.Field{Key: "type", Value: pc.Type})
		registered++
	}

	return registered, errs
}
