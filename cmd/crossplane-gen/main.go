// Package main implements the crossplane-gen CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eric-carlsson/crossplane-gen/pkg/xrd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

var (
	// allGenerators maintains the list of all known generators
	allGenerators = map[string]genall.Generator{
		"xrd": xrd.Generator{},
	}

	// allOutputRules defines the list of all known output rules
	allOutputRules = map[string]genall.OutputRule{
		"dir":       genall.OutputToDirectory(""),
		"none":      genall.OutputToNothing,
		"stdout":    genall.OutputToStdout,
		"artifacts": genall.OutputArtifacts{},
	}

	// optionsRegistry contains all the marker definitions used to process command line options
	optionsRegistry = &markers.Registry{}
)

func init() {
	// Register generator markers
	for genName, gen := range allGenerators {
		defn := markers.Must(markers.MakeDefinition(genName, markers.DescribesPackage, gen))
		if err := optionsRegistry.Register(defn); err != nil {
			panic(err)
		}
		if helpGiver, hasHelp := gen.(genall.HasHelp); hasHelp {
			if help := helpGiver.Help(); help != nil {
				optionsRegistry.AddHelp(defn, help)
			}
		}

		// Register per-generation output rule markers
		for ruleName, rule := range allOutputRules {
			ruleMarker := markers.Must(markers.MakeDefinition(fmt.Sprintf("output:%s:%s", genName, ruleName), markers.DescribesPackage, rule))
			if err := optionsRegistry.Register(ruleMarker); err != nil {
				panic(err)
			}
			if helpGiver, hasHelp := rule.(genall.HasHelp); hasHelp {
				if help := helpGiver.Help(); help != nil {
					optionsRegistry.AddHelp(ruleMarker, help)
				}
			}
		}
	}

	// Register default output rule markers
	for ruleName, rule := range allOutputRules {
		ruleMarker := markers.Must(markers.MakeDefinition("output:"+ruleName, markers.DescribesPackage, rule))
		if err := optionsRegistry.Register(ruleMarker); err != nil {
			panic(err)
		}
		if helpGiver, hasHelp := rule.(genall.HasHelp); hasHelp {
			if help := helpGiver.Help(); help != nil {
				optionsRegistry.AddHelp(ruleMarker, help)
			}
		}
	}

	// Register common options markers (paths, etc)
	if err := genall.RegisterOptionsMarkers(optionsRegistry); err != nil {
		panic(err)
	}
}

func main() {
	cmd := &cobra.Command{
		Use:   "crossplane-gen",
		Short: "Generate Crossplane API resources.",
		Long:  "Generate Crossplane API resources.",
		Example: `	# Generate XRDs for all types under apis/, outputting to /tmp/xrds
	crossplane-gen xrd paths=./apis/... output:dir=/tmp/xrds

	# Generate XRDs and output to stdout
	crossplane-gen xrd paths=./apis/... output:stdout

	# Generate XRDs with custom options
	crossplane-gen xrd:maxDescLen=0 paths=./apis/... output:dir=./config/xrd`,
		RunE: func(_ *cobra.Command, rawOpts []string) error {
			rt, err := genall.FromOptions(optionsRegistry, rawOpts)
			if err != nil {
				return err
			}
			if len(rt.Generators) == 0 {
				return fmt.Errorf("no generators specified")
			}

			if hadErrs := rt.Run(); hadErrs {
				return fmt.Errorf("not all generators ran successfully")
			}
			return nil
		},
		SilenceUsage: true,
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
