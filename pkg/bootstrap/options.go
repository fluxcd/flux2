/*
Copyright 2021 The Flux authors

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

package bootstrap

import (
	"fmt"
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/fluxcd/pkg/git"
	runclient "github.com/fluxcd/pkg/runtime/client"

	"github.com/fluxcd/flux2/v2/pkg/log"
)

type Option interface {
	GitOption
	GitProviderOption
}

func WithBranch(branch string) Option {
	return branchOption(branch)
}

type branchOption string

func (o branchOption) applyGit(b *PlainGitBootstrapper) {
	b.branch = string(o)
}

func (o branchOption) applyGitProvider(b *GitProviderBootstrapper) {
	o.applyGit(b.PlainGitBootstrapper)
}

func WithSignature(name, email string) Option {
	return signatureOption{
		Name:  name,
		Email: email,
	}
}

type signatureOption git.Signature

func (o signatureOption) applyGit(b *PlainGitBootstrapper) {
	if o.Name != "" {
		b.signature.Name = o.Name
	}
	if o.Email != "" {
		b.signature.Email = o.Email
	}
}

func (o signatureOption) applyGitProvider(b *GitProviderBootstrapper) {
	o.applyGit(b.PlainGitBootstrapper)
}

func WithCommitMessageAppendix(appendix string) Option {
	return commitMessageAppendixOption(appendix)
}

type commitMessageAppendixOption string

func (o commitMessageAppendixOption) applyGit(b *PlainGitBootstrapper) {
	b.commitMessageAppendix = string(o)
}

func (o commitMessageAppendixOption) applyGitProvider(b *GitProviderBootstrapper) {
	o.applyGit(b.PlainGitBootstrapper)
}

func WithKubeconfig(rcg genericclioptions.RESTClientGetter, opts *runclient.Options) Option {
	return kubeconfigOption{
		rcg:  rcg,
		opts: opts,
	}
}

type kubeconfigOption struct {
	rcg  genericclioptions.RESTClientGetter
	opts *runclient.Options
}

func (o kubeconfigOption) applyGit(b *PlainGitBootstrapper) {
	b.restClientGetter = o.rcg
	b.restClientOptions = o.opts
}

func (o kubeconfigOption) applyGitProvider(b *GitProviderBootstrapper) {
	o.applyGit(b.PlainGitBootstrapper)
}

func WithLogger(logger log.Logger) Option {
	return loggerOption{logger}
}

type loggerOption struct {
	logger log.Logger
}

func (o loggerOption) applyGit(b *PlainGitBootstrapper) {
	b.logger = o.logger
}

func (o loggerOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.logger = o.logger
}

func WithGitCommitSigning(gpgKeyRing openpgp.EntityList, passphrase, keyID string) Option {
	return gitCommitSigningOption{
		gpgKeyRing:    gpgKeyRing,
		gpgPassphrase: passphrase,
		gpgKeyID:      keyID,
	}
}

type gitCommitSigningOption struct {
	gpgKeyRing    openpgp.EntityList
	gpgPassphrase string
	gpgKeyID      string
}

func (o gitCommitSigningOption) applyGit(b *PlainGitBootstrapper) {
	b.gpgKeyRing = o.gpgKeyRing
	b.gpgPassphrase = o.gpgPassphrase
	b.gpgKeyID = o.gpgKeyID
}

func (o gitCommitSigningOption) applyGitProvider(b *GitProviderBootstrapper) {
	o.applyGit(b.PlainGitBootstrapper)
}

func LoadEntityListFromPath(path string) (openpgp.EntityList, error) {
	if path == "" {
		return nil, nil
	}
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open GPG key ring: %w", err)
	}
	entityList, err := openpgp.ReadKeyRing(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read GPG key ring: %w", err)
	}
	return entityList, nil
}
