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

package flags

import (
	"fmt"
	"strings"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedReceiverTypes = []string{
	notificationv1.GenericReceiver,
	notificationv1.GenericHMACReceiver,
	notificationv1.GitHubReceiver,
	notificationv1.GitLabReceiver,
	notificationv1.BitbucketReceiver,
	notificationv1.HarborReceiver,
	notificationv1.DockerHubReceiver,
	notificationv1.QuayReceiver,
	notificationv1.GCRReceiver,
	notificationv1.NexusReceiver,
	notificationv1.ACRReceiver,
	notificationv1.CDEventsReceiver,
}

type ReceiverType string

func (r *ReceiverType) String() string {
	return string(*r)
}

func (r *ReceiverType) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no receiver type given, please specify %s",
			r.Description())
	}
	if !utils.ContainsItemString(supportedReceiverTypes, str) {
		return fmt.Errorf("receiver type '%s' is not supported, must be one of: %s",
			str, strings.Join(supportedReceiverTypes, ", "))
	}
	*r = ReceiverType(str)
	return nil
}

func (r *ReceiverType) Type() string {
	return strings.Join(supportedReceiverTypes, "|")
}

func (r *ReceiverType) Description() string {
	return "the receiver type"
}
