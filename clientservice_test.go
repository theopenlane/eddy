package eddy

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type serviceTestClient struct {
	name string
}

type serviceTestKey string

func (k serviceTestKey) String() string {
	return string(k)
}

func TestClientServiceGetClientUsesCache(t *testing.T) {
	ctx := context.Background()
	pool := NewClientPool[*serviceTestClient](time.Minute)
	service := NewClientService[*serviceTestClient, string, struct{}](pool)

	var calls int32
	builder := &BuilderFunc[*serviceTestClient, string, struct{}]{
		Type: "service-test",
		Func: func(context.Context, string, struct{}) (*serviceTestClient, error) {
			atomic.AddInt32(&calls, 1)
			return &serviceTestClient{name: "generated"}, nil
		},
	}

	key := serviceTestKey("tenant:service-test")

	first := service.GetClient(ctx, key, builder, "output", struct{}{})
	if !first.IsPresent() {
		t.Fatal("expected first call to build and return a client")
	}

	second := service.GetClient(ctx, key, builder, "output", struct{}{})
	if !second.IsPresent() {
		t.Fatal("expected second call to retrieve client from cache")
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected builder to be called once, called %d times", calls)
	}

	if first.MustGet() != second.MustGet() {
		t.Fatal("expected cached client instances to be identical")
	}
}

func TestClientServiceGetClientBuildError(t *testing.T) {
	ctx := context.Background()
	pool := NewClientPool[*serviceTestClient](time.Minute)
	service := NewClientService[*serviceTestClient, string, struct{}](pool)

	var shouldFail atomic.Bool
	shouldFail.Store(true)

	builder := &BuilderFunc[*serviceTestClient, string, struct{}]{
		Type: "service-test",
		Func: func(context.Context, string, struct{}) (*serviceTestClient, error) {
			if shouldFail.Load() {
				return nil, errors.New("build failed")
			}
			return &serviceTestClient{name: "recovered"}, nil
		},
	}

	key := serviceTestKey("tenant:service-test")

	if client := service.GetClient(ctx, key, builder, "output", struct{}{}); client.IsPresent() {
		t.Fatal("expected builder failure to result in none")
	}

	shouldFail.Store(false)

	success := service.GetClient(ctx, key, builder, "output", struct{}{})
	if !success.IsPresent() {
		t.Fatal("expected builder success to return client")
	}
}

func TestClientServiceGetClientUsesCloneFunctions(t *testing.T) {
	ctx := context.Background()
	pool := NewClientPool[*serviceTestClient](time.Minute)

	type output struct {
		value string
	}

	type config struct {
		flag bool
	}

	var outputClones, configClones int32
	var receivedOutput *output
	var receivedConfig *config

	service := NewClientService(pool,
		WithOutputClone[*serviceTestClient, *output, *config](func(in *output) *output {
			atomic.AddInt32(&outputClones, 1)
			if in == nil {
				return nil
			}
			copy := *in
			return &copy
		}),
		WithConfigClone[*serviceTestClient, *output](func(in *config) *config {
			atomic.AddInt32(&configClones, 1)
			if in == nil {
				return nil
			}
			copy := *in
			return &copy
		}),
	)

	builder := &BuilderFunc[*serviceTestClient, *output, *config]{
		Type: "clone-test",
		Func: func(_ context.Context, out *output, cfg *config) (*serviceTestClient, error) {
			receivedOutput = out
			receivedConfig = cfg
			return &serviceTestClient{name: "clone"}, nil
		},
	}

	origOutput := &output{value: "secret"}
	origConfig := &config{flag: true}

	result := service.GetClient(ctx, serviceTestKey("clone"), builder, origOutput, origConfig)
	if !result.IsPresent() {
		t.Fatal("expected client to be present")
	}

	if atomic.LoadInt32(&outputClones) != 1 {
		t.Fatalf("expected output clone function to be called once, got %d", outputClones)
	}

	if atomic.LoadInt32(&configClones) != 1 {
		t.Fatalf("expected config clone function to be called once, got %d", configClones)
	}

	if receivedOutput == nil || receivedOutput == origOutput {
		t.Fatal("expected cloned output to be passed to builder")
	}

	if receivedConfig == nil || receivedConfig == origConfig {
		t.Fatal("expected cloned config to be passed to builder")
	}

	if receivedOutput.value != origOutput.value {
		t.Fatal("expected cloned output to preserve data")
	}

	if receivedConfig.flag != origConfig.flag {
		t.Fatal("expected cloned config to preserve data")
	}
}

func TestClientServicePool(t *testing.T) {
	pool := NewClientPool[*serviceTestClient](time.Minute)
	service := NewClientService[*serviceTestClient, string, struct{}](pool)

	if service.Pool() != pool {
		t.Fatal("expected Pool to return underlying pool instance")
	}
}
