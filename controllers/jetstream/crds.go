package jetstream

import (
	"context"
	"fmt"
	"time"

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionstyped "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func createStreamCRD(c apiextensionsclientset.Interface) error {
	const plural = "streams"

	crd := &apiextensions.CustomResourceDefinition{
		ObjectMeta: k8smeta.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", plural, apis.GroupName),
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group: apis.GroupName,
			Scope: apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Kind:     "Stream",
				Singular: "stream",
				Plural:   plural,
				ShortNames: []string{
					"str",
				},
			},
			Versions: []apiextensions.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Subresources: &apiextensions.CustomResourceSubresources{
						Status: &apiextensions.CustomResourceSubresourceStatus{},
					},
					Schema: &apiextensions.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensions.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensions.JSONSchemaProps{
								"spec":   getStreamSpecSchema(),
								"status": getStreamStatusSchema(),
							},
						},
					},
					AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
						{
							Name:        "State",
							Type:        "string",
							Description: "The name of the Jetstream Stream",
							JSONPath:    fmt.Sprintf(".status.conditions[?(@.type == %q)].reason", streamReadyCondType),
						},
						{
							Name:        "Stream Name",
							Type:        "string",
							Description: "The name of the Jetstream Stream",
							JSONPath:    ".spec.name",
						},
						{
							Name:        "Subjects",
							Type:        "string",
							Description: "The subjects this Stream produces",
							JSONPath:    ".spec.subjects",
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	crdif := c.ApiextensionsV1().CustomResourceDefinitions()

	var opt k8smeta.CreateOptions
	if _, err := crdif.Create(ctx, crd, opt); k8serrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to create stream CRD: %w", err)
	}

	if err := waitEstablishedCRD(crd.ObjectMeta.Name, crdif); err != nil {
		return fmt.Errorf("CRD failed to become Established: %w", err)
	}
	return nil
}

func getStreamSpecSchema() apiextensions.JSONSchemaProps {
	return apiextensions.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensions.JSONSchemaProps{
			"servers": {
				Description: "A list of Jetstream servers to connect to.",
				Type:        "array",
				MinLength:   int64Ptr(1),
				Items: &apiextensions.JSONSchemaPropsOrArray{
					Schema: &apiextensions.JSONSchemaProps{
						Type:      "string",
						MinLength: int64Ptr(1),
					},
				},
			},
			"name": {
				Description: "A unique name for the Stream. Read-only after first time.",
				Type:        "string",
				Pattern:     `^[^.*>]*$`,
				MinLength:   int64Ptr(0),
			},
			"subjects": {
				Description: "A list of subjects to produce, supports wildcards.",
				Type:        "array",
				MinLength:   int64Ptr(1),
				Items: &apiextensions.JSONSchemaPropsOrArray{
					Schema: &apiextensions.JSONSchemaProps{
						Type:      "string",
						MinLength: int64Ptr(1),
					},
				},
			},
			"storage": {
				Description: "The storage backend to use for the Stream. Read-only after first time.",
				Type:        "string",
				Enum: []apiextensions.JSON{
					{Raw: []byte(`"file"`)},
					{Raw: []byte(`"memory"`)},
				},
				Default: &apiextensions.JSON{Raw: []byte(`"memory"`)},
			},
			"maxAge": {
				Description: "Maximum age of any message in the stream, expressed in Go's time.Duration format. Empty for unlimited.",
				Type:        "string",
				Default:     &apiextensions.JSON{Raw: []byte(`""`)},
			},
		},
	}
}

func getStreamStatusSchema() apiextensions.JSONSchemaProps {
	return apiextensions.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensions.JSONSchemaProps{
			"observedGeneration": {
				Type: "integer",
			},
			"conditions": {
				Type: "array",
				Items: &apiextensions.JSONSchemaPropsOrArray{
					Schema: &apiextensions.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensions.JSONSchemaProps{
							"type": {
								Type: "string",
							},
							"status": {
								Type: "string",
							},
							"lastTransitionTime": {
								Type: "string",
							},
							"reason": {
								Type: "string",
							},
							"message": {
								Type: "string",
							},
						},
					},
				},
			},
		},
	}
}

func waitEstablishedCRD(name string, crdif apiextensionstyped.CustomResourceDefinitionInterface) error {
	return wait.Poll(time.Second, 10*time.Second, func() (bool, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		crd, err := crdif.Get(ctx, name, k8smeta.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, c := range crd.Status.Conditions {
			if c.Type != apiextensions.Established {
				continue
			}
			if c.Status != apiextensions.ConditionTrue {
				continue
			}

			return true, nil
		}

		return false, fmt.Errorf("CRD %q is not established", name)
	})
}

func int64Ptr(n int64) *int64 {
	return &n
}
