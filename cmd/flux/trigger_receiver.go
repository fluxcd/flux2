/*
Copyright 2026 The Flux authors

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
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluxcd/pkg/auth/actionsoidc"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
)

const (
	// genericOIDCReceiver mirrors notificationv1.GenericOIDCReceiver from the
	// upcoming notification-controller release.
	// TODO: Replace it with the constant from the api module once the dependency
	// is bumped.
	genericOIDCReceiver = "generic-oidc"

	// defaultOIDCAudience mirrors notificationv1.DefaultOIDCAudience.
	// TODO: Replace it with the constant from the api module once the dependency
	// is bumped.
	defaultOIDCAudience = "notification-controller"

	// defaultOIDCTokenEnvVar is the environment variable the OIDC token is read
	// from when neither --oidc-provider nor --oidc-token is set.
	defaultOIDCTokenEnvVar = "FLUX_TRIGGER_RECEIVER_OIDC_TOKEN"
)

const (
	oidcProviderGitHub  = "github"
	oidcProviderForgejo = "forgejo"
)

var triggerReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Trigger the webhook of a Receiver",
	Long: `The trigger receiver command sends a request to the incoming webhook of a Receiver.

The command computes the webhook path from the Receiver name, namespace and token,
appends it to the base URL and sends an HTTP POST request with the given payload.
It does not require access to the Kubernetes cluster.`,
	Example: `  # Trigger a generic Receiver
  flux trigger receiver my-receiver \
    --token=my-token \
    --url=https://flux-webhook.example.com

  # Trigger a generic Receiver with a custom JSON payload
  flux trigger receiver my-receiver \
    --token=my-token \
    --url=https://flux-webhook.example.com \
    --payload='{"image":"ghcr.io/org/app:v1.0.0"}'

  # Trigger a generic-hmac Receiver
  flux trigger receiver my-receiver \
    --type=generic-hmac \
    --token=my-token \
    --url=https://flux-webhook.example.com \
    --payload='{"image":"ghcr.io/org/app:v1.0.0"}'

  # Trigger a generic-oidc Receiver from a GitHub Actions workflow.
  # The job needs 'permissions: id-token: write'. The OIDC token is fetched
  # automatically and the receiver token is not used by this type.
  flux trigger receiver my-receiver \
    --type=generic-oidc \
    --oidc-provider=github \
    --url=https://flux-webhook.example.com

  # Trigger a generic-oidc Receiver from a GitHub Actions workflow with a custom OIDC audience
  flux trigger receiver my-receiver \
    --type=generic-oidc \
    --oidc-provider=github \
    --oidc-audience=my-flux-instance \
    --url=https://flux-webhook.example.com

  # Trigger a generic-oidc Receiver from a Forgejo Actions workflow
  flux trigger receiver my-receiver \
    --type=generic-oidc \
    --oidc-provider=forgejo \
    --url=https://flux-webhook.example.com

  # Trigger a generic-oidc Receiver from a GitLab CI/CD job, reading the OIDC
  # token from an id_token environment variable defined in the job spec.
  flux trigger receiver my-receiver \
    --type=generic-oidc \
    --oidc-token="${MY_ID_TOKEN}" \
    --url=https://flux-webhook.example.com

  # Trigger a generic-oidc Receiver from a GitLab CI/CD job, reading the OIDC
  # token from the default FLUX_TRIGGER_RECEIVER_OIDC_TOKEN environment variable,
  # e.g. defined as:
  #   job:
  #     id_tokens:
  #       FLUX_TRIGGER_RECEIVER_OIDC_TOKEN:
  #         aud: notification-controller
  flux trigger receiver my-receiver \
    --type=generic-oidc \
    --url=https://flux-webhook.example.com

  # Trigger a Receiver in a specific namespace
  flux trigger receiver my-receiver -n apps \
    --token=my-token \
    --url=https://flux-webhook.example.com

  # Trigger a Receiver in the namespace of the current kubeconfig context
  flux trigger receiver my-receiver \
    --ns-follows-kube-context \
    --token=my-token \
    --url=https://flux-webhook.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: triggerReceiverCmdRun,
}

type triggerReceiverFlags struct {
	token        string
	url          string
	receiverType string
	oidcProvider string
	oidcToken    string
	oidcAudience string
	payload      string
	retries      int
	retryDelay   time.Duration
}

var triggerReceiverArgs triggerReceiverFlags

func init() {
	triggerReceiverCmd.Flags().StringVar(&triggerReceiverArgs.token, "token", "",
		"the Receiver token, required for all types except generic-oidc where it must not be set")
	triggerReceiverCmd.Flags().StringVar(&triggerReceiverArgs.url, "url", "",
		"the base URL of the notification-controller webhook receiver, may contain a base path")
	triggerReceiverCmd.Flags().StringVar(&triggerReceiverArgs.receiverType, "type", notificationv1.GenericReceiver,
		fmt.Sprintf("the Receiver type, one of: %s, %s, %s",
			notificationv1.GenericReceiver, notificationv1.GenericHMACReceiver, genericOIDCReceiver))
	triggerReceiverCmd.Flags().StringVar(&triggerReceiverArgs.oidcProvider, "oidc-provider", "",
		fmt.Sprintf("the OIDC provider to fetch the token from, one of: %s, %s (generic-oidc only, mutually exclusive with --oidc-token)",
			oidcProviderGitHub, oidcProviderForgejo))
	triggerReceiverCmd.Flags().StringVar(&triggerReceiverArgs.oidcToken, "oidc-token", "",
		fmt.Sprintf("the OIDC token to authenticate the request (generic-oidc only, mutually exclusive with --oidc-provider); defaults to the %s environment variable", defaultOIDCTokenEnvVar))
	triggerReceiverCmd.Flags().StringVar(&triggerReceiverArgs.oidcAudience, "oidc-audience", "",
		fmt.Sprintf("the audience of the OIDC token to fetch (requires --oidc-provider); defaults to %q", defaultOIDCAudience))
	triggerReceiverCmd.Flags().StringVar(&triggerReceiverArgs.payload, "payload", "{}",
		"the JSON payload to send in the request body")
	triggerReceiverCmd.Flags().IntVar(&triggerReceiverArgs.retries, "retries", 10,
		"the number of times to retry on connection errors or retryable HTTP status codes (404, 408, 429, 5xx); set to 0 to disable")
	triggerReceiverCmd.Flags().DurationVar(&triggerReceiverArgs.retryDelay, "retry-delay", 10*time.Second,
		"the delay between retries")

	triggerCmd.AddCommand(triggerReceiverCmd)
}

func triggerReceiverCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if triggerReceiverArgs.url == "" {
		return fmt.Errorf("--url is required")
	}

	if err := validateTriggerReceiverArgs(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	// For generic-oidc the Receiver has no secretRef, so the webhook path is
	// salted with an empty token. For all other types the token is required.
	pathToken := triggerReceiverArgs.token
	if triggerReceiverArgs.receiverType == genericOIDCReceiver {
		pathToken = ""
	}

	receiver := &notificationv1.Receiver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
		},
	}
	webhookURL := strings.TrimRight(triggerReceiverArgs.url, "/") + receiver.GetWebhookPath(pathToken)

	payload := []byte(triggerReceiverArgs.payload)

	// Compute the request headers once; the auth material does not change between
	// attempts, so they are applied to a fresh request on each retry.
	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   fmt.Sprintf("flux/v%s", VERSION),
	}
	switch triggerReceiverArgs.receiverType {
	case notificationv1.GenericReceiver:
		// No authentication, the payload is sent as-is.
	case notificationv1.GenericHMACReceiver:
		mac := hmac.New(sha256.New, []byte(triggerReceiverArgs.token))
		mac.Write(payload)
		headers["X-Signature"] = "sha256=" + hex.EncodeToString(mac.Sum(nil))
	case genericOIDCReceiver:
		oidcToken, err := resolveOIDCToken(ctx)
		if err != nil {
			return err
		}
		headers["Authorization"] = "Bearer " + oidcToken
	}

	// send performs a single attempt. It reports retryable=true for transient
	// failures (connection errors and retryable HTTP status codes) so the caller
	// can retry; permanent failures (e.g. authentication or validation errors)
	// report retryable=false and fail immediately.
	send := func() (retryable bool, err error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
		if err != nil {
			return false, fmt.Errorf("unable to create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return true, fmt.Errorf("request to %s failed: %w", webhookURL, err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			statusErr := fmt.Errorf("request to %s failed with status %s", webhookURL, resp.Status)
			if msg := strings.TrimSpace(string(body)); msg != "" {
				statusErr = fmt.Errorf("request to %s failed with status %s: %s", webhookURL, resp.Status, msg)
			}
			return isRetryableStatus(resp.StatusCode), statusErr
		}
		return false, nil
	}

	logger.Actionf("triggering Receiver %s/%s", *kubeconfigArgs.Namespace, name)
	for attempt := 0; ; attempt++ {
		retryable, err := send()
		if err == nil {
			logger.Successf("Receiver %s/%s triggered", *kubeconfigArgs.Namespace, name)
			return nil
		}
		if !retryable || attempt >= triggerReceiverArgs.retries {
			return err
		}
		logger.Waitingf("%s; retrying in %s (%d/%d)",
			err, triggerReceiverArgs.retryDelay, attempt+1, triggerReceiverArgs.retries)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(triggerReceiverArgs.retryDelay):
		}
	}
}

// isRetryableStatus reports whether an HTTP status returned by the webhook
// receiver is worth retrying. 404 is included because the Receiver's webhook
// path may not be registered yet right after the notification-controller starts
// or while the Receiver reconciles.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusNotFound, // 404
		http.StatusRequestTimeout,      // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// validateTriggerReceiverArgs validates the receiver type and the combination of
// token and OIDC flags.
func validateTriggerReceiverArgs() error {
	isOIDC := triggerReceiverArgs.receiverType == genericOIDCReceiver

	switch triggerReceiverArgs.receiverType {
	case notificationv1.GenericReceiver, notificationv1.GenericHMACReceiver, genericOIDCReceiver:
	default:
		return fmt.Errorf("invalid --type %q, must be one of: %s, %s, %s",
			triggerReceiverArgs.receiverType,
			notificationv1.GenericReceiver, notificationv1.GenericHMACReceiver, genericOIDCReceiver)
	}

	if !isOIDC {
		if triggerReceiverArgs.token == "" {
			return fmt.Errorf("--token is required for --type=%s", triggerReceiverArgs.receiverType)
		}
		if triggerReceiverArgs.oidcProvider != "" || triggerReceiverArgs.oidcToken != "" || triggerReceiverArgs.oidcAudience != "" {
			return fmt.Errorf("--oidc-provider, --oidc-token and --oidc-audience can only be set for --type=%s", genericOIDCReceiver)
		}
		return nil
	}

	// generic-oidc.
	if triggerReceiverArgs.token != "" {
		return fmt.Errorf("--token must not be set for --type=%s, the Receiver of this type has no secret", genericOIDCReceiver)
	}
	if triggerReceiverArgs.oidcProvider != "" && triggerReceiverArgs.oidcToken != "" {
		return fmt.Errorf("--oidc-provider and --oidc-token are mutually exclusive")
	}
	if triggerReceiverArgs.oidcProvider != "" {
		switch triggerReceiverArgs.oidcProvider {
		case oidcProviderGitHub, oidcProviderForgejo:
		default:
			return fmt.Errorf("invalid --oidc-provider %q, must be one of: %s, %s",
				triggerReceiverArgs.oidcProvider, oidcProviderGitHub, oidcProviderForgejo)
		}
	}
	if triggerReceiverArgs.oidcAudience != "" && triggerReceiverArgs.oidcProvider == "" {
		return fmt.Errorf("--oidc-audience can only be set together with --oidc-provider")
	}

	return nil
}

// resolveOIDCToken returns the OIDC token used to authenticate the request,
// either by fetching it from the configured provider or by reading it from the
// --oidc-token flag or the default environment variable.
func resolveOIDCToken(ctx context.Context) (string, error) {
	switch {
	case triggerReceiverArgs.oidcProvider != "":
		audience := triggerReceiverArgs.oidcAudience
		if audience == "" {
			audience = defaultOIDCAudience
		}
		// GitHub and Forgejo Actions expose the same token request endpoint.
		token, _, err := actionsoidc.FetchToken(ctx, audience)
		return token, err
	case triggerReceiverArgs.oidcToken != "":
		return triggerReceiverArgs.oidcToken, nil
	default:
		token := os.Getenv(defaultOIDCTokenEnvVar)
		if token == "" {
			return "", fmt.Errorf("no OIDC token provided: set --oidc-provider, --oidc-token or the %s environment variable", defaultOIDCTokenEnvVar)
		}
		return token, nil
	}
}
