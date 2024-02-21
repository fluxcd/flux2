# Flux ARM64 GitHub runners

The Flux ARM64 end-to-end tests run on Equinix Metal instances provisioned with Docker and GitHub self-hosted runners.

## Current instances

| Repository                  | Runner           | Instance       | Location      |
|-----------------------------|------------------|----------------|---------------|
| flux2                       | equinix-arm-dc-1 | flux-arm-dc-01 | Washington DC |
| flux2                       | equinix-arm-dc-2 | flux-arm-dc-01 | Washington DC |
| flux2                       | equinix-arm-da-1 | flux-arm-da-01 | Dallas        |
| flux2                       | equinix-arm-da-2 | flux-arm-da-01 | Dallas        |
| flux-benchmark              | equinix-arm-dc-1 | flux-arm-dc-01 | Washington DC |
| flux-benchmark              | equinix-arm-da-1 | flux-arm-da-01 | Dallas        |
| source-controller           | equinix-arm-dc-1 | flux-arm-dc-01 | Washington DC |
| source-controller           | equinix-arm-da-1 | flux-arm-da-01 | Dallas        |
| image-automation-controller | equinix-arm-dc-1 | flux-arm-dc-01 | Washington DC |
| image-automation-controller | equinix-arm-da-1 | flux-arm-da-01 | Dallas        |

Instance spec:
- Ampere Altra Q80-30 80-core processor @ 2.8GHz
- 2 x 960GB NVME
- 256GB RAM
- 2 x 25Gbps

## Instance setup

In order to add a new runner to the GitHub Actions pool,
first create a server on Equinix with the following configuration:
- Type: `c3.large.arm64`
- OS: `Ubuntu 22.04 LTS`

### Install prerequisites

- SSH into a newly created instance
```shell
ssh root@<instance-public-IP>
``` 

- Create the ubuntu user
```shell
adduser ubuntu
usermod -aG sudo ubuntu
su - ubuntu
```

- Create the prerequisites dir
```shell
mkdir -p prereq && cd prereq
```

- Download the prerequisites script
```shell
curl -sL https://raw.githubusercontent.com/fluxcd/flux2/main/.github/runners/prereq.sh > prereq.sh \
  && chmod +x ./prereq.sh
```

- Install the prerequisites
```shell
sudo ./prereq.sh
```

### Install runners

- Retrieve the GitHub runner token from the repository [settings page](https://github.com/fluxcd/flux2/settings/actions/runners/new?arch=arm64&os=linux)

- Create two directories `flux2-01`, `flux2-02`

- In each dir run:
```shell
curl -sL https://raw.githubusercontent.com/fluxcd/flux2/main/.github/runners/runner-setup.sh > runner-setup.sh \
  && chmod +x ./runner-setup.sh

./runner-setup.sh equinix-arm-<NUMBER> <TOKEN> <REPO>
```

- Reboot the instance
```shell
sudo reboot
```

- Navigate to the GitHub repository [runners page](https://github.com/fluxcd/flux2/settings/actions/runners) and check the runner status
