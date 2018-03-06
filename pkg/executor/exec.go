package executor

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kubernauts/parameterizer/pkg/parameterizer"
)

// Run executes the Parameterizer resource's transformation
// steps as defined in the apply sub-resource.
func Run(p parameterizer.Resource) (err error) {
	// we create a temporary manifest file with all
	// the necessary settings in there
	mf, mc, err := createmanifest(p)
	if err != nil {
		return err
	}
	mfn := mf.Name()
	defer func() {
		e := os.Remove(mfn)
		if e != nil {
			fmt.Printf("Couldn't clean up temporary manifest %v", mfn)
		}
	}()
	fmt.Printf("Using manifest:\n%v\n", mc)
	cmd := []string{"create", "-f", mfn}
	fmt.Printf("Executing command: %v\n", strings.Join(cmd, " "))
	res, err := kubectl(true, cmd[0], cmd[1:]...)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", res)
	time.Sleep(1 * time.Minute)
	res, err = kubectl(true, "get", "po", "-a")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", res)
	res, err = kubectl(true, "logs", "pexecutor")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", res)
	res, err = kubectl(true, "delete", "po", "pexecutor", "--force")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", res)
	return nil
}

func createmanifest(p parameterizer.Resource) (*os.File, string, error) {
	content := []byte(`apiVersion: v1
kind: Pod
metadata:
  name: pexecutor
spec: 
  initContainers:
  - name: resinput
    image: alpine:3.7
    command: ["sh", "-c", "wget -O /work/charts.zip ` + p.Spec.Resources[0].Source.URLs[0] + ` && unzip /work/charts.zip" ]
    volumeMounts:
    - name: pmr
      mountPath: "/work"
  containers:
  - name: ` + p.Spec.Apply[0].Name + `
    image: ` + p.Spec.Apply[0].Image + `
    command: ["sh", "-c", "` + p.Spec.Apply[0].Commands[0] + `" ]
    volumeMounts:
    - name: pmr 
      mountPath: "/work" 
  volumes:
  - name: pmr
    hostPath:
      path: /tmp/pmr`)
	tmpf, err := ioutil.TempFile("/tmp", "krm")
	if err != nil {
		return nil, "", err
	}
	if _, err := tmpf.Write(content); err != nil {
		return nil, "", err
	}
	if err := tmpf.Close(); err != nil {
		return nil, "", err
	}
	return tmpf, string(content), nil
}

func buildcmds(cmds []string) string {
	var res string
	for _, cmd := range cmds {
		wocmd := strings.Split(cmd, " ")[1:]
		res += strings.Join(wocmd, " ")
	}
	return res
}

func kubectl(withstderr bool, cmd string, args ...string) (string, error) {
	kubectlbin, err := executecmd(false, "which", "kubectl")
	if err != nil {
		return "", err
	}
	all := append([]string{cmd}, args...)
	result, err := executecmd(withstderr, kubectlbin, all...)
	if err != nil {
		return "", err
	}
	return result, nil
}

func executecmd(withstderr bool, cmd string, args ...string) (string, error) {
	result := ""
	var out bytes.Buffer
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	if withstderr {
		c.Stderr = os.Stderr
	}
	c.Stdout = &out
	err := c.Run()
	if err != nil {
		return result, err
	}
	result = strings.TrimSpace(out.String())
	return result, nil
}
