/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
)

var createSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Create or update a Bucket source",
	Long: withPreviewNote(`The create source bucket command generates a Bucket resource and waits for it to be downloaded.
For Buckets with static authentication, the credentials are stored in a Kubernetes secret.`),
	Example: `  # Create a source for a Bucket using static authentication
  flux create source bucket podinfo \
	--bucket-name=podinfo \
    --endpoint=minio.minio.svc.cluster.local:9000 \
	--insecure=true \
	--access-key=myaccesskey \
	--secret-key=mysecretkey \
    --interval=10m

  # Create a source for an Amazon S3 Bucket using IAM authentication
  flux create source bucket podinfo \
	--bucket-name=podinfo \
	--provider=aws \
    --endpoint=s3.amazonaws.com \
	--region=us-east-1 \
    --interval=10m`,
	RunE: createSourceBucketCmdRun,
}

type sourceBucketFlags struct {
	name        string
	provider    flags.SourceBucketProvider
	endpoint    string
	accessKey   string
	secretKey   string
	region      string
	insecure    bool
	secretRef   string
	ignorePaths []string
}

var sourceBucketArgs = newSourceBucketFlags()

func init() {
	createSourceBucketCmd.Flags().Var(&sourceBucketArgs.provider, "provider", sourceBucketArgs.provider.Description())
	createSourceBucketCmd.Flags().StringVar(&sourceBucketArgs.name, "bucket-name", "", "the bucket name")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketArgs.endpoint, "endpoint", "", "the bucket endpoint address")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketArgs.accessKey, "access-key", "", "the bucket access key")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketArgs.secretKey, "secret-key", "", "the bucket secret key")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketArgs.region, "region", "", "the bucket region")
	createSourceBucketCmd.Flags().BoolVar(&sourceBucketArgs.insecure, "insecure", false, "for when connecting to a non-TLS S3 HTTP endpoint")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketArgs.secretRef, "secret-ref", "", "the name of an existing secret containing credentials")
	createSourceBucketCmd.Flags().StringSliceVar(&sourceBucketArgs.ignorePaths, "ignore-paths", nil, "set paths to ignore in bucket resource (can specify multiple paths with commas: path1,path2)")

	createSourceCmd.AddCommand(createSourceBucketCmd)
}

func newSourceBucketFlags() sourceBucketFlags {
	return sourceBucketFlags{
		provider: flags.SourceBucketProvider(sourcev1.GenericBucketProvider),
	}
}

func createSourceBucketCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if sourceBucketArgs.name == "" {
		return fmt.Errorf("bucket-name is required")
	}

	if sourceBucketArgs.endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	var ignorePaths *string
	if len(sourceBucketArgs.ignorePaths) > 0 {
		ignorePathsStr := strings.Join(sourceBucketArgs.ignorePaths, "\n")
		ignorePaths = &ignorePathsStr
	}

	bucket := &sourcev1.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: sourcev1.BucketSpec{
			BucketName: sourceBucketArgs.name,
			Provider:   sourceBucketArgs.provider.String(),
			Insecure:   sourceBucketArgs.insecure,
			Endpoint:   sourceBucketArgs.endpoint,
			Region:     sourceBucketArgs.region,
			Interval: metav1.Duration{
				Duration: createArgs.interval,
			},
			Ignore: ignorePaths,
		},
	}

	if createSourceArgs.fetchTimeout > 0 {
		bucket.Spec.Timeout = &metav1.Duration{Duration: createSourceArgs.fetchTimeout}
	}

	if sourceBucketArgs.secretRef != "" {
		bucket.Spec.SecretRef = &meta.LocalObjectReference{
			Name: sourceBucketArgs.secretRef,
		}
	}

	if createArgs.export {
		return printExport(exportBucket(bucket))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	logger.Generatef("generating Bucket source")

	if sourceBucketArgs.secretRef == "" {
		secretName := fmt.Sprintf("bucket-%s", name)

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: *kubeconfigArgs.Namespace,
				Labels:    sourceLabels,
			},
			StringData: map[string]string{},
		}

		if sourceBucketArgs.accessKey != "" && sourceBucketArgs.secretKey != "" {
			secret.StringData["accesskey"] = sourceBucketArgs.accessKey
			secret.StringData["secretkey"] = sourceBucketArgs.secretKey
		}

		if len(secret.StringData) > 0 {
			logger.Actionf("applying secret with the bucket credentials")
			if err := upsertSecret(ctx, kubeClient, secret); err != nil {
				return err
			}
			bucket.Spec.SecretRef = &meta.LocalObjectReference{
				Name: secretName,
			}
			logger.Successf("authentication configured")
		}
	}

	logger.Actionf("applying Bucket source")
	namespacedName, err := upsertBucket(ctx, kubeClient, bucket)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for Bucket source reconciliation")
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		isObjectReadyConditionFunc(kubeClient, namespacedName, bucket)); err != nil {
		return err
	}
	logger.Successf("Bucket source reconciliation completed")

	if bucket.Status.Artifact == nil {
		return fmt.Errorf("Bucket source reconciliation but no artifact was found")
	}
	logger.Successf("fetched revision: %s", bucket.Status.Artifact.Revision)
	return nil
}

func upsertBucket(ctx context.Context, kubeClient client.Client,
	bucket *sourcev1.Bucket) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: bucket.GetNamespace(),
		Name:      bucket.GetName(),
	}

	var existing sourcev1.Bucket
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, bucket); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("Bucket source created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = bucket.Labels
	existing.Spec = bucket.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	bucket = &existing
	logger.Successf("Bucket source updated")
	return namespacedName, nil
}
