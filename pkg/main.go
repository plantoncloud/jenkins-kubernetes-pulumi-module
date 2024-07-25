package pkg

import (
	"github.com/pkg/errors"
	"github.com/plantoncloud/jenkins-kubernetes-pulumi-module/pkg/locals"
	"github.com/plantoncloud/jenkins-kubernetes-pulumi-module/pkg/outputs"
	"github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/code2cloud/v1/kubernetes/jenkinskubernetes/model"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/provider/kubernetes/pulumikubernetesprovider"
	kubernetescorev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ResourceStack struct {
	Input  *model.JenkinsKubernetesStackInput
	Labels map[string]string
}

func (s *ResourceStack) Resources(ctx *pulumi.Context) error {
	//create kubernetes-provider from the credential in the stack-input
	kubernetesProvider, err := pulumikubernetesprovider.GetWithKubernetesClusterCredential(ctx,
		s.Input.KubernetesClusterCredential)
	if err != nil {
		return errors.Wrap(err, "failed to setup gcp provider")
	}

	//create namespace resource
	createdNamespace, err := kubernetescorev1.NewNamespace(ctx, locals.Namespace, &kubernetescorev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaPtrInput(&metav1.ObjectMetaArgs{
			Name:   pulumi.String(locals.Namespace),
			Labels: pulumi.ToStringMap(s.Labels),
		}),
	}, pulumi.Timeouts(&pulumi.CustomTimeouts{Create: "5s", Update: "5s", Delete: "5s"}),
		pulumi.Provider(kubernetesProvider))
	if err != nil {
		return errors.Wrapf(err, "failed to create %s namespace", locals.Namespace)
	}

	//export name of the namespace
	ctx.Export(outputs.Namespace, createdNamespace.Metadata.Name())

	//create admin-password secret
	createdAdminPasswordSecret, err := adminCredentials(ctx, createdNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to create admin password resources")
	}

	//install the jenkins helm-chart
	if err := helmChart(ctx, createdNamespace, createdAdminPasswordSecret); err != nil {
		return errors.Wrap(err, "failed to create helm-chart resources")
	}

	//no ingress resources required when ingress is not enabled
	if !locals.JenkinsKubernetes.Spec.Ingress.IsEnabled {
		return nil
	}

	//create istio-ingress resources
	if err := istioIngress(ctx, createdNamespace, s.Labels); err != nil {
		return errors.Wrap(err, "failed to create istio ingress resources")
	}

	return nil
}
