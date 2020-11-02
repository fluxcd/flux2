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
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var createSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Create or update a Bucket source",
	Long: `
The create source bucket command generates a Bucket resource and waits for it to be downloaded.
For Buckets with static authentication, the credentials are stored in a Kubernetes secret.`,
	Example: `  # Create a source from a Buckets using static authentication
  flux create source bucket podinfo \
	--bucket-name=podinfo \
    --endpoint=minio.minio.svc.cluster.local:9000 \
	--insecure=true \
	--access-key=myaccesskey \
	--secret-key=mysecretkey \
    --interval=10m

  # Create a source from an Amazon S3 Bucket using IAM authentication
  flux create source bucket podinfo \
	--bucket-name=podinfo \
	--provider=aws \
    --endpoint=s3.amazonaws.com \
	--region=us-east-1 \
    --interval=10m
`,
	RunE: createSourceBucketCmdRun,
}

var (
	sourceBucketName      string
	sourceBucketProvider  = flags.SourceBucketProvider(sourcev1.GenericBucketProvider)
	sourceBucketEndpoint  string
	sourceBucketAccessKey string
	sourceBucketSecretKey string
	sourceBucketRegion    string
	sourceBucketInsecure  bool
	sourceBucketSecretRef string
)

func init() {
	createSourceBucketCmd.Flags().Var(&sourceBucketProvider, "provider", sourceBucketProvider.Description())
	createSourceBucketCmd.Flags().StringVar(&sourceBucketName, "bucket-name", "", "the bucket name")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketEndpoint, "endpoint", "", "the bucket endpoint address")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketAccessKey, "access-key", "", "the bucket access key")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketSecretKey, "secret-key", "", "the bucket secret key")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketRegion, "region", "", "the bucket region")
	createSourceBucketCmd.Flags().BoolVar(&sourceBucketInsecure, "insecure", false, "for when connecting to a non-TLS S3 HTTP endpoint")
	createSourceBucketCmd.Flags().StringVar(&sourceBucketSecretRef, "secret-ref", "", "the name of an existing secret containing credentials")

	createSourceCmd.AddCommand(createSourceBucketCmd)
}

func createSourceBucketCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Bucket source name is required")
	}
	name := args[0]

	if sourceBucketName == "" {
		return fmt.Errorf("bucket-name is required")
	}

	if sourceBucketEndpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	bucket := &sourcev1.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    sourceLabels,
		},
		Spec: sourcev1.BucketSpec{
			BucketName: sourceBucketName,
			Provider:   sourceBucketProvider.String(),
			Insecure:   sourceBucketInsecure,
			Endpoint:   sourceBucketEndpoint,
			Region:     sourceBucketRegion,
			Interval: metav1.Duration{
				Duration: interval,
			},
		},
	}
	if sourceHelmSecretRef != "" {
		bucket.Spec.SecretRef = &corev1.LocalObjectReference{
			Name: sourceBucketSecretRef,
		}
	}

	if export {
		return exportBucket(*bucket)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	logger.Generatef("generating Bucket source")

	if sourceBucketSecretRef == "" {
		secretName := fmt.Sprintf("bucket-%s", name)

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			StringData: map[string]string{},
		}

		if sourceBucketAccessKey != "" && sourceBucketSecretKey != "" {
			secret.StringData["accesskey"] = sourceBucketAccessKey
			secret.StringData["secretkey"] = sourceBucketSecretKey
		}

		if len(secret.StringData) > 0 {
			logger.Actionf("applying secret with the bucket credentials")
			if err := upsertSecret(ctx, kubeClient, secret); err != nil {
				return err
			}
			bucket.Spec.SecretRef = &corev1.LocalObjectReference{
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
	if err := wait.PollImmediate(pollInterval, timeout,
		isBucketReady(ctx, kubeClient, namespacedName, bucket)); err != nil {
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
