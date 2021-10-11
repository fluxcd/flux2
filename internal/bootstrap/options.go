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
	"github.com/fluxcd/flux2/internal/bootstrap/git"
	"github.com/fluxcd/flux2/pkg/log"
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

func WithAuthor(name, email string) Option {
	return authorOption{
		Name:  name,
		Email: email,
	}
}

type authorOption git.Author

func (o authorOption) applyGit(b *PlainGitBootstrapper) {
	if o.Name != "" {
		b.author.Name = o.Name
	}
	if o.Email != "" {
		b.author.Email = o.Email
	}
}

func (o authorOption) applyGitProvider(b *GitProviderBootstrapper) {
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

func WithKubeconfig(kubeconfig, kubecontext string) Option {
	return kubeconfigOption{
		kubeconfig:  kubeconfig,
		kubecontext: kubecontext,
	}
}

type kubeconfigOption struct {
	kubeconfig  string
	kubecontext string
}

func (o kubeconfigOption) applyGit(b *PlainGitBootstrapper) {
	b.kubeconfig = o.kubeconfig
	b.kubecontext = o.kubecontext
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

func WithGitCommitSigning(path, passphrase, keyID string) Option {
	return gitCommitSigningOption{
		gpgKeyRingPath: path,
		gpgPassphrase:  passphrase,
		gpgKeyID:       keyID,
	}
}

type gitCommitSigningOption struct {
	gpgKeyRingPath string
	gpgPassphrase  string
	gpgKeyID       string
}

func (o gitCommitSigningOption) applyGit(b *PlainGitBootstrapper) {
	b.gpgKeyRingPath = o.gpgKeyRingPath
	b.gpgPassphrase = o.gpgPassphrase
	b.gpgKeyID = o.gpgKeyID
}

func (o gitCommitSigningOption) applyGitProvider(b *GitProviderBootstrapper) {
	o.applyGit(b.PlainGitBootstrapper)
}
