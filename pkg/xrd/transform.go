package xrd

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	xpv2 "github.com/crossplane/crossplane/v2/apis/apiextensions/v2"
)

// CRDToXRDv2 converts a Kubernetes CustomResourceDefinition to a Crossplane CompositeResourceDefinition (v2).
func CRDToXRDv2(crd *apiextensionsv1.CustomResourceDefinition) (*xpv2.CompositeResourceDefinition, error) {
	xrd := &xpv2.CompositeResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: xpv2.SchemeGroupVersion.String(),
			Kind:       xpv2.CompositeResourceDefinitionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        crd.Name,
			Annotations: crd.Annotations,
			Labels:      crd.Labels,
		},
		Spec: xpv2.CompositeResourceDefinitionSpec{
			Group:    crd.Spec.Group,
			Names:    crd.Spec.Names,
			Versions: convertVersions(crd.Spec.Versions),
		},
	}

	// Set conversion strategy if present
	if crd.Spec.Conversion != nil {
		xrd.Spec.Conversion = crd.Spec.Conversion
	}

	return xrd, nil
}

func convertVersions(crdVersions []apiextensionsv1.CustomResourceDefinitionVersion) []xpv2.CompositeResourceDefinitionVersion {
	xrdVersions := make([]xpv2.CompositeResourceDefinitionVersion, 0, len(crdVersions))

	for _, crdVer := range crdVersions {
		xrdVer := xpv2.CompositeResourceDefinitionVersion{
			Name:          crdVer.Name,
			Referenceable: crdVer.Storage, // Map storage=true to referenceable=true
			Served:        crdVer.Served,
			Schema:        convertSchema(crdVer.Schema),
		}

		// Convert additional printer columns if present
		if len(crdVer.AdditionalPrinterColumns) > 0 {
			xrdVer.AdditionalPrinterColumns = crdVer.AdditionalPrinterColumns
		}

		// Convert deprecated flag
		if crdVer.Deprecated {
			xrdVer.Deprecated = &crdVer.Deprecated
			xrdVer.DeprecationWarning = crdVer.DeprecationWarning
		}

		xrdVersions = append(xrdVersions, xrdVer)
	}

	return xrdVersions
}

func convertSchema(crdSchema *apiextensionsv1.CustomResourceValidation) *xpv2.CompositeResourceValidation {
	if crdSchema == nil || crdSchema.OpenAPIV3Schema == nil {
		return nil
	}

	// Marshal schema, as CompositeResourceValidation requires runtime.RawExtension
	raw, err := json.Marshal(crdSchema.OpenAPIV3Schema)
	if err != nil {
		return nil
	}

	return &xpv2.CompositeResourceValidation{
		OpenAPIV3Schema: runtime.RawExtension{Raw: raw},
	}
}
