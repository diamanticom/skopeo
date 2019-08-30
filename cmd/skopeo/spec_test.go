package main

import (
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
)

func TestSpecGen(t *testing.T) {
	s := new(Spec)
	s.Kind = SpecKind
	s.ApiVersion = SpecApiVersionV1Beta1
	s.Policy = new(Policy)
	s.Policy.Enforcing = true
	s.Policy.Tag = "latest"
	s.Images = []string{
		"docker://docker.io/library/ubuntu",
		"docker://docker.io/library/fedora",
		"docker://docker.io/library/alpine",
	}

	yb, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(yb))
}
