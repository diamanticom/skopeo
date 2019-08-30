package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Spec defines spec used to reconcile local state with remote
type Spec struct {
	// Kind is the spec descriptor
	Kind string
	// ApiVersion is the spec API
	ApiVersion string
	// Policy is the reconciliation policy
	Policy *Policy
	// Files is a list of files to reconcile
	Images []string
}

// Policy is to enforce presence of a certain tag
type Policy struct {
	Tag       string
	Enforcing bool
}

// Response is to unmarshal output from the query
type Response struct {
	RepoTags []string
}

type specApplyOptions struct {
	global *globalOptions
	image  *imageOptions
	raw    bool // Output the raw manifest instead of parsing information about the image
	config bool // Output the raw config blob instead of parsing information about the image
}

func specGen(args []string, stdout io.Writer) error {
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
		return err
	}

	_, _ = fmt.Fprint(stdout, string(yb))
	return nil
}

func specCmd(global *globalOptions) cli.Command {
	_, sharedOpts := sharedImageFlags()
	_, imageOpts := imageFlags(global, sharedOpts, "", "")
	applyOpts := specApplyOptions{
		global: global,
		image:  imageOpts,
	}
	return cli.Command{
		Name: "spec",
		Subcommands: []cli.Command{
			{
				Name:   "gen",
				Usage:  "generate example spec",
				Action: commandAction(specGen),
			},
			{
				Name:      "apply",
				Usage:     "apply a spec",
				ArgsUsage: "SPEC_FILE",
				Action:    commandAction(applyOpts.run),
			},
		},
	}
}

func (opts *specApplyOptions) run(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("Exactly one argument expected")
	}
	specFileName := args[0]
	b, err := ioutil.ReadFile(specFileName)
	if err != nil {
		return err
	}

	spec := new(Spec)
	if err := yaml.Unmarshal(b, spec); err != nil {
		return err
	}

	if spec.Kind != SpecKind {
		return fmt.Errorf("invalid spec, incorrect kind")
	}

	if spec.ApiVersion != SpecApiVersionV1Beta1 {
		return fmt.Errorf("invalid spec, incorrect api version")
	}

	for _, image := range spec.Images {
		bb := new(bytes.Buffer)
		bw := bufio.NewWriter(bb)

		iopts := inspectOptions{
			global: opts.global,
			image:  opts.image,
		}

		if err := iopts.run([]string{image}, bw); err != nil {
			return err
		}

		if err := bw.Flush(); err != nil {
			return err
		}

		response := new(Response)

		if err := json.Unmarshal(bb.Bytes(), response); err != nil {
			return err
		}

		_, _ = fmt.Fprintln(stdout, image)
		tagFound := false
		if spec.Policy.Enforcing {
			for _, tag := range response.RepoTags {
				if tag == spec.Policy.Tag {
					tagFound = true
					break
				}
			}
		}

		if spec.Policy.Enforcing && !tagFound {
			return fmt.Errorf("expected tag:%s, which was not found upstream for image:%s",
				spec.Policy.Tag, image)
		}
	}

	return nil
}
