package main

import (
	"testing"
)

func TestTemplateRendering(t *testing.T) {
	manifests, err := template("test/nginx", "nginx", "test/nginx/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	for _, manifest := range manifests {
		if kind := manifest["kind"]; kind == nil {
			t.Fatal("expected kind to be string, got nil")
		}

		if kind := manifest["apiVersion"]; kind == nil {
			t.Fatal("expected apiVersion to be string, got nil")
		}

		metadata := manifest["metadata"]
		if metadata == nil {
			t.Fatal("expected metadata to exist, got nil")
		}

		if name := metadata.(map[string]interface{})["name"]; name == nil {
			t.Fatal("expected metadat.name to exist, got nil")
		}

	}
}
