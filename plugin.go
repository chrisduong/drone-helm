package main

import (
	"fmt"
	"github.com/alecthomas/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"strings"
)

var HELM_BIN = "/bin/helm"
var KUBECONFIG = "/root/.kube/kubeconfig"
var CONFIG = "/root/.kube/config"

type (
	// Config maps the params we need to run Helm
	Config struct {
		APIServer     string   `json:"api_server"`
		Token         string   `json:"token"`
		HelmCommand   []string `json:"helm_command"`
		SkipTLSVerify bool     `json:"tls_skip_verify"`
		Namespace     string   `json:"namespace"`
		Release       string   `json:"release"`
		Chart         string   `json:"chart"`
		Values        string   `json:"values"`
		Debug         bool     `json:"debug"`
		DryRun        bool     `json:"dry_run"`
		Secrets       []string `json:"secrets"`
		Prefix        string   `json:"prefix"`
		TillerNs      string   `json:"tiller_ns"`
	}
	// Plugin default
	Plugin struct {
		Config Config
	}
)

func setHelmHelp(p *Plugin) {
	p.Config.HelmCommand = []string{""}
}
func setDeleteEventCommand(p *Plugin) {
	upgrade := make([]string, 2)
	upgrade[0] = "delete"
	upgrade[1] = p.Config.Release

	p.Config.HelmCommand = upgrade
}

func setPushEventCommand(p *Plugin) {
	upgrade := make([]string, 2)
	upgrade[0] = "upgrade"
	upgrade[1] = "--install"

	if p.Config.Release != "" {
		upgrade = append(upgrade, p.Config.Release)
	}
	upgrade = append(upgrade, p.Config.Chart)
	if p.Config.Values != "" {
		upgrade = append(upgrade, "--set")
		upgrade = append(upgrade, p.Config.Values)
	}
	if p.Config.Namespace != "" {
		upgrade = append(upgrade, "--namespace")
		upgrade = append(upgrade, p.Config.Namespace)
	}
	if p.Config.DryRun {
		upgrade = append(upgrade, "--dry-run")
	}
	if p.Config.Debug {
		upgrade = append(upgrade, "--debug")
	}
	p.Config.HelmCommand = upgrade

}

func setHelmCommand(p *Plugin) {
	buildEvent := os.Getenv("DRONE_BUILD_EVENT")
	switch buildEvent {
	case "push":
		setPushEventCommand(p)
	case "delete":
		setDeleteEventCommand(p)
	default:
		setHelmHelp(p)
	}

}

func doHelmInit(p *Plugin) []string {
	init := make([]string, 1)
	init[0] = "init"
	if p.Config.TillerNs != "" {
		init = append(init, "--tiller-namespace")
		init = append(init, p.Config.TillerNs)
	}
	return init

}

// Exec default method
func (p *Plugin) Exec() error {
	if p.Config.APIServer == "" {
		return fmt.Errorf("Error: API Server is needed to deploy.")
	}
	if p.Config.Token == "" {
		return fmt.Errorf("Error: Token is needed to deploy.")
	}
	initialiseKubeconfig(&p.Config, KUBECONFIG, CONFIG)

	if p.Config.Debug {
		p.debug()
	}

	init := doHelmInit(p)
	err := runCommand(init)
	if err != nil {
		return fmt.Errorf("Error running helm comand: " + strings.Join(init[:], " "))
	}
	setHelmCommand(p)

	if p.Config.Debug {
		log.Println("helm command: " + strings.Join(p.Config.HelmCommand[:], " "))
	}
	err = runCommand(p.Config.HelmCommand)
	if err != nil {
		return fmt.Errorf("Error running helm comand: " + strings.Join(p.Config.HelmCommand[:], " "))
	}
	return nil
}

func initialiseKubeconfig(params *Config, source string, target string) error {
	t, _ := template.ParseFiles(source)
	f, err := os.Create(target)
	err = t.Execute(f, params)
	f.Close()
	return err
}

func runCommand(params []string) error {
	cmd := new(exec.Cmd)
	cmd = exec.Command(HELM_BIN, params...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return err
}

func (p *Plugin) debug() {
	fmt.Println(p)
	// debug env vars
	for _, e := range os.Environ() {
		fmt.Println("-Var:--", e)
	}
	// debug plugin obj
	fmt.Printf("Api server: %s \n", p.Config.APIServer)
	fmt.Printf("Values: %s \n", p.Config.Values)
	fmt.Printf("Values: %s \n", p.Config.Secrets)

	kubeconfig, err := ioutil.ReadFile(KUBECONFIG)
	if err == nil {
		fmt.Println(string(kubeconfig))
	}
	config, err := ioutil.ReadFile(CONFIG)
	if err == nil {
		fmt.Println(string(config))
	}

}
