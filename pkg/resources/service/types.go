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

package service

// Service represents knative service structure
type Service struct {
	Annotations    map[string]string
	BuildArgs      []string
	BuildTimeout   string
	Concurrency    int
	Env            []string
	EnvSecrets     []string
	Labels         []string
	Name           string
	Namespace      string
	PullPolicy     string
	Registry       string
	Revision       string
	ResultImageTag string
	RegistrySecret string // Does not belong to the service, need to be deleted
	Runtime        string // Originally knative/buildtemplate, but now also tekton/task
	Source         string
	Cronjob        struct {
		Schedule string
		Data     string
	}
}


type registryAuths struct {
	Auths registry
}

type credentials struct {
	Username string
	Password string
}

type registry map[string]credentials
