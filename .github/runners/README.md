# Flux GitHub runners

How to provision GitHub Actions self-hosted runners for Flux conformance testing.

## ARM64 Instance specs

In order to add a new runner to the GitHub Actions pool,
first create an instance on Oracle Cloud with the following configuration:
- OS: Canonical Ubuntu 20.04
- Shape: VM.Standard.A1.Flex
- OCPU Count: 2 
- Memory (GB): 12
- Network Bandwidth (Gbps): 2
- Local Disk: Block Storage Only  

Note that the instance image source must be **Canonical Ubuntu** instead of the default Oracle Linux.

## ARM64 Instance setup

- SSH into a newly created instance
```shell
ssh ubuntu@<instance-public-IP>
``` 
- Create the action runner dir
```shell
mkdir -p actions-runner && cd actions-runner
```
- Download the provisioning script
```shell
curl -sL https://raw.githubusercontent.com/fluxcd/flux2/main/.github/runners/arm64.sh > arm64.sh \
  && chmod +x ./arm64.sh
```
- Retrieve the GitHub runner token from the repository [settings page](https://github.com/fluxcd/flux2/settings/actions/runners/new?arch=arm64&os=linux)
- Run the provisioning script passing the token as the first argument
```shell
sudo ./arm64.sh <TOKEN>
```
- Reboot the instance
```shell
sudo reboot
```  
- Navigate to the GitHub repository [runners page](https://github.com/fluxcd/flux2/settings/actions/runners) and check the runner status
