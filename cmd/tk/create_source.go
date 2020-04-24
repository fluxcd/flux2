package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"text/template"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var createSourceCmd = &cobra.Command{
	Use:   "source [name]",
	Short: "Create source resource",
	Long: `
The create source command generates a source.fluxcd.io resource and waits for it to sync.
If a Git repository is specified, it will create a SSH deploy key.`,
	Example: `  create source podinfo --git-url ssh://git@github.com/stefanprodan/podinfo-deploy`,
	RunE:    createSourceCmdRun,
}

var (
	sourceGitURL    string
	sourceGitBranch string
)

func init() {
	createSourceCmd.Flags().StringVar(&sourceGitURL, "git-url", "", "git SSH address, in the format ssh://git@host/org/repository")
	createSourceCmd.Flags().StringVar(&sourceGitBranch, "git-branch", "master", "git branch")

	createCmd.AddCommand(createSourceCmd)
}

func createSourceCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
	}
	name := args[0]

	if sourceGitURL == "" {
		return fmt.Errorf("git-url is required")
	}

	tmpDir, err := ioutil.TempDir("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	u, err := url.Parse(sourceGitURL)
	if err != nil {
		return fmt.Errorf("git URL parse failed: %w", err)
	}

	fmt.Println(`✚`, "generating host key for", u.Host)

	keyscan := fmt.Sprintf("ssh-keyscan %s > %s/known_hosts", u.Host, tmpDir)
	if output, err := execCommand(keyscan); err != nil {
		return fmt.Errorf("ssh-keyscan failed: %s", output)
	}

	fmt.Println(`✚`, "generating deploy key")

	keygen := fmt.Sprintf("ssh-keygen -b 2048 -t rsa -f %s/identity -q -N \"\"", tmpDir)
	if output, err := execCommand(keygen); err != nil {
		return fmt.Errorf("ssh-keygen failed: %s", output)
	}

	deployKey, err := execCommand(fmt.Sprintf("cat %s/identity.pub", tmpDir))
	if err != nil {
		return fmt.Errorf("unable to read identity.pub: %w", err)
	}

	fmt.Print(deployKey)
	prompt := promptui.Prompt{
		Label:     "Have you added the deploy key to your repository",
		IsConfirm: true,
	}
	if _, err := prompt.Run(); err != nil {
		fmt.Println(`✗`, "aborting")
		return nil
	}

	fmt.Println(`✚`, "saving deploy key")
	files := fmt.Sprintf("--from-file=%s/identity --from-file=%s/identity.pub --from-file=%s/known_hosts",
		tmpDir, tmpDir, tmpDir)
	secret := fmt.Sprintf("kubectl -n %s create secret generic %s %s --dry-run=client -oyaml | kubectl apply -f-",
		namespace, name, files)
	if output, err := execCommand(secret); err != nil {
		return fmt.Errorf("kubectl create secret failed: %s", output)
	} else {
		fmt.Print(output)
	}

	fmt.Println(`✚`, "generating source resource")

	t, err := template.New("tmpl").Parse(gitSource)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	source := struct {
		Name      string
		Namespace string
		GitURL    string
		Interval  string
	}{
		Name:      name,
		Namespace: namespace,
		GitURL:    sourceGitURL,
		Interval:  interval,
	}

	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := t.Execute(writer, source); err != nil {
		return fmt.Errorf("template execution failed: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("source flush failed: %w", err)
	}

	if output, err := execCommand(fmt.Sprintf("echo '%s' | kubectl apply -f-", data.String())); err != nil {
		return fmt.Errorf("kubectl create source failed: %s", output)
	} else {
		fmt.Print(output)
	}

	fmt.Println(`✚`, "waiting for source sync")
	if output, err := execCommand(fmt.Sprintf(
		"kubectl -n %s wait gitrepository/%s --for=condition=ready --timeout=1m",
		namespace, name)); err != nil {
		return fmt.Errorf("source sync failed: %s", output)
	} else {
		fmt.Print(output)
	}

	return nil
}

var gitSource = `---
apiVersion: source.fluxcd.io/v1alpha1
kind: GitRepository
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  interval: {{.Interval}}
  url: {{.GitURL}}
  secretRef:
    name: {{.Name}}
`
