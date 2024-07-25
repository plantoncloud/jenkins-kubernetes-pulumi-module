package pkg

import (
	"github.com/pkg/errors"
	"github.com/plantoncloud/jenkins-kubernetes-pulumi-module/pkg/locals"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/provider/kubernetes/containerresources"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/provider/kubernetes/helm/mergemaps"
	kubernetescorev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func helmChart(ctx *pulumi.Context,
	createdNamespace *kubernetescorev1.Namespace,
	createdAdminPasswordSecret *kubernetescorev1.Secret) error {

	// https://github.com/jenkinsci/helm-charts/blob/main/charts/jenkins/values.yaml
	var helmValues = pulumi.Map{
		"fullnameOverride": pulumi.String(locals.JenkinsKubernetes.Metadata.Name),
		"controller": pulumi.Map{
			"image": pulumi.Map{
				"tag": pulumi.String(vars.JenkinsDockerImageTag),
			},
			"resources": containerresources.ConvertToPulumiMap(locals.JenkinsKubernetes.Spec.Container.Resources),
			"admin": pulumi.Map{
				"passwordKey":    pulumi.String(vars.JenkinsAdminPasswordSecretKey),
				"existingSecret": createdAdminPasswordSecret.Metadata.Name(),
			},
		},
	}

	//merge extra helm values provided in the spec with base values
	mergemaps.MergeMapToPulumiMap(helmValues, locals.JenkinsKubernetes.Spec.HelmValues)

	//install jenkins-helm chart
	_, err := helmv3.NewChart(ctx,
		locals.JenkinsKubernetes.Metadata.Id,
		helmv3.ChartArgs{
			Chart:     pulumi.String(vars.HelmChartName),
			Version:   pulumi.String(vars.HelmChartVersion),
			Namespace: createdNamespace.Metadata.Name().Elem(),
			Values:    helmValues,
			//if you need to add the repository, you can specify `repo url`:
			FetchArgs: helmv3.FetchArgs{
				Repo: pulumi.String(vars.HelmChartRepoUrl),
			},
		}, pulumi.Parent(createdNamespace))
	if err != nil {
		return errors.Wrap(err, "failed to create helm chart")
	}

	return nil
}
