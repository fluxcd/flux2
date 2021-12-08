# Flux ARM64 GitHub runners

The Flux ARM64 end-to-end tests run on Equinix instances provisioned with Docker and GitHub self-hosted runners.

## Current instances

| Runner        | Instance            | Region |
|---------------|---------------------|--------|
| equinix-arm-1 | flux-equinix-arm-01 | AMS1   |
| equinix-arm-2 | flux-equinix-arm-01 | AMS1   |
| equinix-arm-3 | flux-equinix-arm-01 | AMS1   |
| equinix-arm-4 | flux-equinix-arm-02 | DFW2   |
| equinix-arm-5 | flux-equinix-arm-02 | DFW2   |
| equinix-arm-6 | flux-equinix-arm-02 | DFW2   |

## Instance setup

In order to add a new runner to the GitHub Actions pool,
first create a server on Equinix with the following configuration:
- Type: c2.large.arm
- OS: Ubuntu 20.04

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

- Create 3 directories `runner1`, `runner2`, `runner3`

- In each dir run:
```shell
curl -sL https://raw.githubusercontent.com/fluxcd/flux2/main/.github/runners/runner-setup.sh > runner-setup.sh \
  && chmod +x ./runner-setup.sh

./runner-setup.sh equinix-arm-<NUMBER> <TOKEN>
```

- Reboot the instance
```shell
sudo reboot
```

- Navigate to the GitHub repository [runners page](https://github.com/fluxcd/flux2/settings/actions/runners) and check the runner status
