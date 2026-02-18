// Package xrd contains utilities for generating Crossplane Composite Resource Definitions (XRD).
// It uses controller-tools crd package internally to first generate corresponding CRDs, and then
// converts then to XRDs.
package xrd

import (
	"fmt"
	"strings"

	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

// Generator generates Crossplane CompositeResourceDefinition (XRD) objects.
type Generator struct {
	// IgnoreUnexportedFields indicates that we should skip unexported fields.
	IgnoreUnexportedFields *bool `marker:",optional"`

	// AllowDangerousTypes allows types which are usually omitted from CRD generation.
	AllowDangerousTypes *bool `marker:",optional"`

	// MaxDescLen specifies the maximum description length for fields in XRD's OpenAPI schema.
	MaxDescLen *int `marker:",optional"`

	// GenerateEmbeddedObjectMeta specifies if any embedded ObjectMeta in the XRD should be generated
	GenerateEmbeddedObjectMeta *bool `marker:",optional"`

	// HeaderFile specifies the header text (e.g. license) to prepend to generated files.
	HeaderFile string `marker:",optional"`

	// Year specifies the year to substitute for " YEAR" in the header file.
	Year string `marker:",optional"`
}

// CheckFilter returns the generator's node filter.
func (Generator) CheckFilter() loader.NodeFilter {
	return crd.Generator{}.CheckFilter()
}

// RegisterMarkers registers all markers needed by this generator.
func (g Generator) RegisterMarkers(into *markers.Registry) error {
	// Register all CRD markers since we need them for parsing
	return crd.Generator{}.RegisterMarkers(into)
}

// removeEmptyStatus removes the status field from the XRD output
func removeXRDStatus(obj map[string]any) error {
	delete(obj, "status")
	return nil
}

// Generate generates XRD resources.
func (g Generator) Generate(ctx *genall.GenerationContext) error {
	parser := &crd.Parser{
		Collector: ctx.Collector,
		Checker:   ctx.Checker,
	}

	// Set parser options from generator config
	if g.IgnoreUnexportedFields != nil {
		parser.IgnoreUnexportedFields = *g.IgnoreUnexportedFields
	}
	if g.AllowDangerousTypes != nil {
		parser.AllowDangerousTypes = *g.AllowDangerousTypes
	}
	if g.GenerateEmbeddedObjectMeta != nil {
		parser.GenerateEmbeddedObjectMeta = *g.GenerateEmbeddedObjectMeta
	}

	crd.AddKnownTypes(parser)
	for _, root := range ctx.Roots {
		parser.NeedPackage(root)
	}

	metav1Pkg := crd.FindMetav1(ctx.Roots)
	if metav1Pkg == nil {
		return nil
	}

	kubeKinds := crd.FindKubeKinds(parser, metav1Pkg)
	if len(kubeKinds) == 0 {
		return nil
	}

	var headerText string

	if g.HeaderFile != "" {
		headerBytes, err := ctx.ReadFile(g.HeaderFile)
		if err != nil {
			return err
		}
		headerText = string(headerBytes)
	}
	headerText = strings.ReplaceAll(headerText, " YEAR", " "+g.Year)

	// Generate XRDs for each kind
	for _, groupKind := range kubeKinds {
		parser.NeedCRDFor(groupKind, g.MaxDescLen)
		crdRaw := parser.CustomResourceDefinitions[groupKind]

		// Validate storage version
		hasStorage := false
		for _, ver := range crdRaw.Spec.Versions {
			if ver.Storage {
				hasStorage = true
				break
			}
		}
		if !hasStorage {
			return fmt.Errorf("XRD %s.%s must have at least one version with +kubebuilder:storageversion marker",
				groupKind.Kind, groupKind.Group)
		}

		// Convert CRD to XRD
		xrd, err := CRDToXRDv2(&crdRaw)
		if err != nil {
			return fmt.Errorf("failed to convert CRD to XRD for %s: %w", groupKind, err)
		}

		fileName := fmt.Sprintf("%s_%s.yaml", crdRaw.Spec.Group, crdRaw.Spec.Names.Plural)
		if err := ctx.WriteYAML(fileName, headerText, []any{xrd}, genall.WithTransform(removeXRDStatus)); err != nil {
			return fmt.Errorf("failed to write XRD: %w", err)
		}
	}

	return nil
}
