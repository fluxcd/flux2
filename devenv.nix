{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/basics/
  env.GREET = "Flux2 dev env";

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.kustomize
  ];

  # https://devenv.sh/scripts/
  scripts.hello.exec = "echo Hello in the $GREET";
  scripts.tests.exec = "make test";

  enterShell = ''
    hello
  '';

  # https://devenv.sh/tests/
  enterTest = ''
    tests
  '';

  # https://devenv.sh/services/
  # services.postgres.enable = true;

  # https://devenv.sh/languages/
  # languages.nix.enable = true;
  languages.go.enable = true;

  # https://devenv.sh/pre-commit-hooks/
  # pre-commit.hooks.shellcheck.enable = true;

  # https://devenv.sh/processes/
  # processes.ping.exec = "ping example.com";

  # See full reference at https://devenv.sh/reference/options/
}