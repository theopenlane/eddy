package eddy

import (
	"context"
	"errors"
	"testing"
)

type builderTestClient struct {
	id string
}

func TestBuilderFuncBuild(t *testing.T) {
	ctx := context.Background()
	output := "credentials"
	config := struct{ retries int }{retries: 3}

	expected := &builderTestClient{id: "client-123"}
	builder := &BuilderFunc[*builderTestClient, string, struct{ retries int }]{
		Type: "test-provider",
		Func: func(c context.Context, gotOutput string, gotConfig struct{ retries int }) (*builderTestClient, error) {
			if gotOutput != output {
				t.Fatalf("expected output %q, got %q", output, gotOutput)
			}
			if gotConfig != config {
				t.Fatalf("expected config %#v, got %#v", config, gotConfig)
			}
			if c == nil {
				t.Fatal("context should not be nil")
			}
			return expected, nil
		},
	}

	client, err := builder.Build(ctx, output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client != expected {
		t.Fatalf("expected client %v, got %v", expected, client)
	}

	if builder.ProviderType() != "test-provider" {
		t.Fatalf("expected provider type %q, got %q", "test-provider", builder.ProviderType())
	}
}

func TestBuilderFuncBuildError(t *testing.T) {
	expectedErr := errors.New("failure")
	builder := &BuilderFunc[*builderTestClient, string, struct{}]{
		Type: "error-provider",
		Func: func(context.Context, string, struct{}) (*builderTestClient, error) {
			return nil, expectedErr
		},
	}

	client, err := builder.Build(context.Background(), "output", struct{}{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if client != nil {
		t.Fatalf("expected nil client, got %v", client)
	}
}
