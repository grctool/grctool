// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"fmt"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger(t *testing.T) logger.Logger {
	t.Helper()
	log, err := logger.NewTestLogger()
	require.NoError(t, err)
	return log
}

// stubFactory creates a StubDataProvider with the given name.
func stubFactory(pc config.ProviderConfig, log logger.Logger) (interfaces.DataProvider, error) {
	return testhelpers.NewStubDataProvider(pc.Name), nil
}

// errorFactory always returns an error.
func errorFactory(pc config.ProviderConfig, log logger.Logger) (interfaces.DataProvider, error) {
	return nil, fmt.Errorf("factory error for %s", pc.Name)
}

func TestInitFromConfig_EnabledProvider(t *testing.T) {
	log := testLogger(t)

	// Save and restore global factories
	saved := factories
	factories = map[string]ProviderFactoryFunc{"stub": stubFactory}
	defer func() { factories = saved }()

	cfg := config.ProvidersConfig{
		Providers: []config.ProviderConfig{
			{Name: "my-stub", Type: "stub", Enabled: true},
		},
	}
	reg := NewProviderRegistry()

	count, errs := InitFromConfig(cfg, reg, log)
	assert.Equal(t, 1, count)
	assert.Empty(t, errs)
	assert.True(t, reg.Has("my-stub"))
}

func TestInitFromConfig_DisabledProvider(t *testing.T) {
	log := testLogger(t)

	saved := factories
	factories = map[string]ProviderFactoryFunc{"stub": stubFactory}
	defer func() { factories = saved }()

	cfg := config.ProvidersConfig{
		Providers: []config.ProviderConfig{
			{Name: "disabled-one", Type: "stub", Enabled: false},
		},
	}
	reg := NewProviderRegistry()

	count, errs := InitFromConfig(cfg, reg, log)
	assert.Equal(t, 0, count)
	assert.Empty(t, errs)
	assert.False(t, reg.Has("disabled-one"))
}

func TestInitFromConfig_UnknownType(t *testing.T) {
	log := testLogger(t)

	saved := factories
	factories = map[string]ProviderFactoryFunc{}
	defer func() { factories = saved }()

	cfg := config.ProvidersConfig{
		Providers: []config.ProviderConfig{
			{Name: "mystery", Type: "unknown-type", Enabled: true},
		},
	}
	reg := NewProviderRegistry()

	count, errs := InitFromConfig(cfg, reg, log)
	assert.Equal(t, 0, count)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "no factory registered")
	assert.Contains(t, errs[0].Error(), "unknown-type")
}

func TestInitFromConfig_FactoryError(t *testing.T) {
	log := testLogger(t)

	saved := factories
	factories = map[string]ProviderFactoryFunc{"broken": errorFactory}
	defer func() { factories = saved }()

	cfg := config.ProvidersConfig{
		Providers: []config.ProviderConfig{
			{Name: "broken-provider", Type: "broken", Enabled: true},
		},
	}
	reg := NewProviderRegistry()

	count, errs := InitFromConfig(cfg, reg, log)
	assert.Equal(t, 0, count)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "factory error")
}

func TestInitFromConfig_DuplicateName(t *testing.T) {
	log := testLogger(t)

	saved := factories
	factories = map[string]ProviderFactoryFunc{"stub": stubFactory}
	defer func() { factories = saved }()

	cfg := config.ProvidersConfig{
		Providers: []config.ProviderConfig{
			{Name: "dup", Type: "stub", Enabled: true},
			{Name: "dup", Type: "stub", Enabled: true},
		},
	}
	reg := NewProviderRegistry()

	count, errs := InitFromConfig(cfg, reg, log)
	assert.Equal(t, 1, count) // first succeeds
	require.Len(t, errs, 1)   // second fails
	assert.Contains(t, errs[0].Error(), "already registered")
}

func TestInitFromConfig_MixedProviders(t *testing.T) {
	// Not parallel — mutates global factories map
	log := testLogger(t)

	saved := factories
	factories = map[string]ProviderFactoryFunc{
		"good":   stubFactory,
		"broken": errorFactory,
	}
	defer func() { factories = saved }()

	cfg := config.ProvidersConfig{
		Providers: []config.ProviderConfig{
			{Name: "p1", Type: "good", Enabled: true},
			{Name: "p2", Type: "broken", Enabled: true},
			{Name: "p3", Type: "good", Enabled: false},
			{Name: "p4", Type: "unknown", Enabled: true},
			{Name: "p5", Type: "good", Enabled: true},
		},
	}
	reg := NewProviderRegistry()

	count, errs := InitFromConfig(cfg, reg, log)
	assert.Equal(t, 2, count) // p1 and p5
	assert.Len(t, errs, 2)    // p2 (factory error) + p4 (unknown type)
	assert.True(t, reg.Has("p1"))
	assert.False(t, reg.Has("p2"))
	assert.False(t, reg.Has("p3"))
	assert.False(t, reg.Has("p4"))
	assert.True(t, reg.Has("p5"))
}

func TestInitFromConfig_EmptyConfig(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	reg := NewProviderRegistry()
	count, errs := InitFromConfig(config.ProvidersConfig{}, reg, log)
	assert.Equal(t, 0, count)
	assert.Empty(t, errs)
}

func TestRegisterFactory(t *testing.T) {
	saved := factories
	factories = map[string]ProviderFactoryFunc{}
	defer func() { factories = saved }()

	RegisterFactory("test-type", stubFactory)
	types := RegisteredFactoryTypes()
	assert.Contains(t, types, "test-type")
}
