package generators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CommandGenerator struct {
	commands []string
	dir      string
}

func NewCommandGenerator(commands ...string) *CommandGenerator {
	return &CommandGenerator{
		commands: commands,
	}
}

func (cg *CommandGenerator) SetWorkingDirectory(dir string) {
	cg.dir = dir
}

var _ Generator = &CommandGenerator{}

func (g *CommandGenerator) Generate(ctx context.Context, app *v1alpha1.Application) ([]*unstructured.Unstructured, error) {
	cmd := exec.Command(g.commands[0], g.commands[1:]...)
	cmd.Dir = g.dir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	defer stdin.Close()

	stderr := bytes.NewBuffer(nil)
	cmd.Stderr = stderr

	go func() {
		json.NewEncoder(stdin).Encode(app)

		stdin.Close()
	}()

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var execErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer stdout.Close()

		execErr = cmd.Start()

		if execErr != nil {
			return
		}

		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case <-ctx.Done():
				cmd.Process.Kill()
			case <-done:
			}
		}()

		execErr = cmd.Wait()
	}()

	res, err := yaml.ParseUnstructuredAll(stdout)
	wg.Wait()

	if execErr != nil {
		err = execErr
	}

	if err != nil {
		if stderr.Len() == 0 {
			return nil, err
		}

		return nil, fmt.Errorf("%+v: %+v", execErr, stderr.String())
	}

	if len(res) == 0 && stderr.Len() != 0 {
		return nil, fmt.Errorf("%s", stderr.String())
	}

	return res, nil
}
