// Copyright 2019 TriggerMesh, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/triggermesh/tm/pkg/client"
	"github.com/triggermesh/tm/pkg/file"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kind              = "Task"
	api               = "tekton.dev/v1alpha1"
	uploadDoneTrigger = ".uploadIsDone"
)

// Deploy accepts path (local or URL) to tekton Task manifest and installs it
func (t *Task) Deploy(clientset *client.ConfigSet) (*tekton.Task, error) {
	if !file.IsLocal(t.File) {
		clientset.Log.Debugf("cannot find %q locally, downloading\n", t.File)
		path, err := file.Download(t.File)
		if err != nil {
			return nil, fmt.Errorf("task not found: %s", err)
		}
		t.File = path
	}

	task, err := t.readYAML()
	if err != nil {
		return nil, err
	}

	// sometimes, if input param type is not set, string value causing the error
	// so we're explicitly setting "string" param type
	if inputs := task.TaskSpec().Inputs; inputs != nil {
		for k, v := range inputs.Params {
			if v.Type == "" {
				inputs.Params[k].Type = tekton.ParamTypeString
			}
		}
	}

	task.SetNamespace(t.Namespace)
	if t.GenerateName != "" {
		task.SetName("")
		task.SetGenerateName(t.GenerateName)
	} else if t.Name != "" {
		task.SetName(t.Name)
	}

	if t.RegistrySecret != "" {
		clientset.Log.Debugf("setting registry secret %q for task \"%s/%s\"\n", t.RegistrySecret, task.GetNamespace(), task.GetName())
		t.setupEnv(task)
		t.setupVolume(task)
	}

	if t.FromLocalSource {
		clientset.Log.Debugf("adding source uploading step to task \"%s/%s\"\n", task.GetNamespace(), task.GetGenerateName())
		task.Spec.Steps = append([]tekton.Step{t.customStep()}, task.Spec.Steps...)
		if task.Spec.Inputs != nil {
			task.Spec.Inputs.Resources = []tekton.TaskResource{}
		}
	}
	if client.Dry {
		return task, nil
	}
	return t.CreateOrUpdate(task, clientset)
}

// Clone installs a copy of provided tekton task object with generated name suffix
func (t *Task) Clone(clientset *client.ConfigSet, task *tekton.Task) (*tekton.Task, error) {
	task.Kind = kind
	task.APIVersion = api
	task.SetGenerateName(task.GetName() + "-")
	task.SetName("")
	task.SetResourceVersion("")
	if t.RegistrySecret != "" {
		t.setupEnv(task)
		t.setupVolume(task)
	}
	if t.FromLocalSource {
		clientset.Log.Debugf("adding source uploading step to task \"%s/%s\" clone\n", task.GetNamespace(), task.GetGenerateName())
		task.Spec.Steps = append([]tekton.Step{t.customStep()}, task.Spec.Steps...)
		if task.Spec.Inputs != nil {
			task.Spec.Inputs.Resources = []tekton.TaskResource{}
		}
	}
	if client.Dry {
		return task, nil
	}
	return t.CreateOrUpdate(task, clientset)
}

func (t *Task) customStep() tekton.Step {
	return tekton.Step{
		Container: corev1.Container{
			Name:    "sources-receiver",
			Image:   "busybox",
			Command: []string{"sh"},
			Args: []string{"-c", fmt.Sprintf(`
				while [ ! -f %s ]; do 
					sleep 1; 
				done; 
				sync;
				mkdir -p /workspace/workspace;
				mv /home/*/* /workspace/workspace/;
				if [[ $? != 0 ]]; then
					mv /home/* /workspace/workspace/;
				fi
				ls -lah /workspace/workspace;
				sync;`,
				uploadDoneTrigger)},
		},
	}
}

func (t *Task) readYAML() (*tekton.Task, error) {
	var res tekton.Task
	yamlFile, err := ioutil.ReadFile(t.File)
	if err != nil {
		return &res, err
	}
	return &res, yaml.Unmarshal(yamlFile, &res)
}

// CreateOrUpdate creates new tekton Task object or updates existing one
func (t *Task) CreateOrUpdate(task *tekton.Task, clientset *client.ConfigSet) (*tekton.Task, error) {
	if task.TypeMeta.Kind != kind {
		return nil, fmt.Errorf("Object Kind mismatch: got %q, want %q", task.TypeMeta.Kind, kind)
	}
	if task.TypeMeta.APIVersion != api {
		return nil, fmt.Errorf("Object API mismatch: got %q, want %q", task.TypeMeta.APIVersion, api)
	}

	if task.GetGenerateName() != "" {
		return clientset.TektonPipelines.TektonV1alpha1().Tasks(t.Namespace).Create(task)
	}

	taskObj, err := clientset.TektonPipelines.TektonV1alpha1().Tasks(t.Namespace).Create(task)
	if k8sErrors.IsAlreadyExists(err) {
		clientset.Log.Debugf("task %q is already exist, updating\n", task.GetName())
		if taskObj, err = clientset.TektonPipelines.TektonV1alpha1().Tasks(t.Namespace).Get(task.ObjectMeta.Name, metav1.GetOptions{}); err != nil {
			return nil, err
		}
		task.ObjectMeta.ResourceVersion = taskObj.GetResourceVersion()
		taskObj, err = clientset.TektonPipelines.TektonV1alpha1().Tasks(t.Namespace).Update(task)
	}
	return taskObj, err
}

// SetOwner updates tekton Task object with provided owner reference
func (t *Task) SetOwner(clientset *client.ConfigSet, owner metav1.OwnerReference) error {
	task, err := clientset.TektonPipelines.TektonV1alpha1().Tasks(t.Namespace).Get(t.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	clientset.Log.Debugf("setting task \"%s/%s\" owner to %s/%s\n", task.GetNamespace(), task.GetName(), owner.Kind, owner.Name)
	task.SetOwnerReferences([]metav1.OwnerReference{owner})
	_, err = clientset.TektonPipelines.TektonV1alpha1().Tasks(t.Namespace).Update(task)
	return err
}

func (t *Task) setupEnv(task *tekton.Task) {
	for i, container := range task.Spec.Steps {
		envs := append(container.Env, corev1.EnvVar{
			Name:  "DOCKER_CONFIG",
			Value: "/" + t.RegistrySecret,
		})
		task.Spec.Steps[i].Env = envs
	}
}

func (t *Task) setupVolume(task *tekton.Task) {
	task.Spec.Volumes = []corev1.Volume{
		{
			Name: t.RegistrySecret,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: t.RegistrySecret,
				},
			},
		},
	}
	for i, step := range task.Spec.Steps {
		mounts := append(step.VolumeMounts, corev1.VolumeMount{
			Name:      t.RegistrySecret,
			MountPath: "/" + t.RegistrySecret,
			ReadOnly:  true,
		})
		task.Spec.Steps[i].VolumeMounts = mounts
	}
}
