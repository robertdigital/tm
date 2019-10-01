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

package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/triggermesh/tm/pkg/client"
)

func newDeployCmd(clientset *client.ConfigSet) *cobra.Command {
	deployCmd := &cobra.Command{
		Use:     "deploy",
		Aliases: []string{"create"},
		Short:   "Deploy knative resource",
		Run: func(cmd *cobra.Command, args []string) {
			s.Namespace = client.Namespace
			s.Registry = client.Registry
			if err := s.DeployYAML(yaml, args, concurrency, clientset); err != nil {
				log.Fatal(err)
			}
		},
	}

	deployCmd.Flags().StringVarP(&yaml, "from", "f", "serverless.yaml", "Deploy functions defined in yaml")
	deployCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 3, "Number on concurrent deployment threads")

	deployCmd.AddCommand(cmdDeployService(clientset))
	deployCmd.AddCommand(cmdDeployChannel(clientset))
	deployCmd.AddCommand(cmdDeployBuild(clientset))
	deployCmd.AddCommand(cmdDeployBuildTemplate(clientset))
	deployCmd.AddCommand(cmdDeployTask(clientset))
	deployCmd.AddCommand(cmdDeployTaskRun(clientset))
	deployCmd.AddCommand(cmdDeployPipelineResource(clientset))
	return deployCmd
}

func cmdDeployService(clientset *client.ConfigSet) *cobra.Command {
	deployServiceCmd := &cobra.Command{
		Use:     "service",
		Aliases: []string{"services", "svc"},
		Short:   "Deploy knative service",
		Args:    cobra.ExactArgs(1),
		Example: "tm deploy service foo -f gcr.io/google-samples/hello-app:1.0",
		Run: func(cmd *cobra.Command, args []string) {
			s.Name = args[0]
			s.Namespace = client.Namespace
			s.Registry = client.Registry
			output, err := s.Deploy(clientset)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(output)
		},
	}
	// kept for back compatibility
	deployServiceCmd.Flags().StringVar(&s.Source, "from-path", "", "Deprecated, use `-f` flag instead")
	deployServiceCmd.Flags().StringVar(&s.Source, "from-image", "", "Deprecated, use `-f` flag instead")
	deployServiceCmd.Flags().StringVar(&s.Source, "from-source", "", "Deprecated, use `-f` flag instead")
	deployServiceCmd.Flags().StringVar(&s.Runtime, "build-template", "", "Deprecated, use `--runtime` flag instead")

	deployServiceCmd.Flags().StringVarP(&s.Source, "from", "f", "", "Service source to deploy: local folder with sources, git repository or docker image")
	deployServiceCmd.Flags().StringVar(&s.Revision, "revision", "master", "Git revision (branch, tag, commit SHA or ref)")
	deployServiceCmd.Flags().StringVar(&s.Runtime, "runtime", "", "Existing buildtemplate name, local path or URL to buildtemplate yaml file")
	deployServiceCmd.Flags().StringVar(&s.RegistrySecret, "registry-secret", "", "Name of k8s secret to use in buildtemplate as registry auth json")
	// deployServiceCmd.Flags().StringVar(&s.ResultImageTag, "tag", "latest", "Image tag to build")
	// deployServiceCmd.Flags().StringVar(&s.PullPolicy, "image-pull-policy", "Always", "Image pull policy")
	deployServiceCmd.Flags().StringVar(&s.BuildTimeout, "build-timeout", "10m", "Service image build timeout")
	deployServiceCmd.Flags().IntVar(&s.Concurrency, "concurrency", 0, "Number of concurrent events per container: 0 - multiple events, 1 - single event, N - particular number of events")
	deployServiceCmd.Flags().StringSliceVar(&s.BuildArgs, "build-argument", []string{}, "Buildtemplate arguments")
	deployServiceCmd.Flags().StringSliceVar(&s.EnvSecrets, "env-secret", []string{}, "Name of k8s secrets to populate pod environment variables")
	deployServiceCmd.Flags().StringSliceVarP(&s.Labels, "label", "l", []string{}, "Service labels")
	deployServiceCmd.Flags().StringToStringVarP(&s.Annotations, "annotation", "a", map[string]string{}, "Revision template annotations")
	deployServiceCmd.Flags().StringSliceVarP(&s.Env, "env", "e", []string{}, "Environment variables of the service, eg. `--env foo=bar`")
	return deployServiceCmd
}

func cmdDeployBuild(clientset *client.ConfigSet) *cobra.Command {
	deployBuildCmd := &cobra.Command{
		Use:     "build",
		Aliases: []string{"builds"},
		Args:    cobra.ExactArgs(1),
		Short:   "Deploy knative build",
		Example: "tm deploy build foo-builder --source git-repo --buildtemplate kaniko --args IMAGE=knative-local-registry:5000/foo-image",
		Run: func(cmd *cobra.Command, args []string) {
			b.Name = args[0]
			b.Namespace = client.Namespace
			if _, err := b.Deploy("", clientset); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Build created")
		},
	}

	deployBuildCmd.Flags().StringVar(&b.Source, "source", "", "Git URL or local path to get sources from")
	deployBuildCmd.Flags().StringVar(&b.Revision, "revision", "master", "Git source revision")
	deployBuildCmd.Flags().StringVar(&b.Buildtemplate, "buildtemplate", "", "Buildtemplate name to use with build")
	deployBuildCmd.Flags().StringSliceVar(&b.Args, "args", []string{}, "Build arguments")
	deployBuildCmd.MarkFlagRequired("source")
	return deployBuildCmd
}

func cmdDeployBuildTemplate(clientset *client.ConfigSet) *cobra.Command {
	deployBuildTemplateCmd := &cobra.Command{
		Use:     "buildtemplate",
		Aliases: []string{"buildtemplates", "bldtmpl"},
		Short:   "Deploy knative build template",
		Example: "tm -n default deploy buildtemplate -f https://raw.githubusercontent.com/triggermesh/nodejs-runtime/master/knative-build-template.yaml",
		Run: func(cmd *cobra.Command, args []string) {
			bt.Namespace = client.Namespace
			if _, err := bt.Deploy(clientset); err != nil {
				log.Fatal(err)
			}
		},
	}

	deployBuildTemplateCmd.Flags().StringVarP(&bt.File, "from", "f", "", "Local path or URL to buildtemplate yaml file")
	deployBuildTemplateCmd.Flags().StringVar(&bt.RegistrySecret, "credentials", "", "Name of k8s secret to use in buildtemplate as registry auth json")
	return deployBuildTemplateCmd
}

func cmdDeployChannel(clientset *client.ConfigSet) *cobra.Command {
	deployChannelCmd := &cobra.Command{
		Use:     "channel",
		Aliases: []string{"channels"},
		Args:    cobra.ExactArgs(1),
		Short:   "Deploy knative eventing in-memory CRD channel",
		Run: func(cmd *cobra.Command, args []string) {
			c.Name = args[0]
			c.Namespace = client.Namespace
			if err := c.Deploy(clientset); err != nil {
				log.Fatal(err)
			}
		},
	}

	// only InMemoryChannel kind of channels available now
	// deployChannelCmd.Flags().StringVarP(&c.Kind, "kind", "k", "InMemoryChannel", "Channel kind")
	return deployChannelCmd
}

func cmdDeployTask(clientset *client.ConfigSet) *cobra.Command {
	deployTaskCmd := &cobra.Command{
		Use:     "task",
		Aliases: []string{"tasks"},
		Short:   "Deploy tekton Task object",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				t.Name = args[0]
			}
			t.Namespace = client.Namespace
			if _, err := t.Deploy(clientset); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Task installed")
		},
	}
	deployTaskCmd.Flags().StringVarP(&t.File, "file", "f", "", "Task yaml manifest path")
	return deployTaskCmd
}

func cmdDeployTaskRun(clientset *client.ConfigSet) *cobra.Command {
	deployTaskRunCmd := &cobra.Command{
		Use:     "taskrun",
		Aliases: []string{"taskruns"},
		Short:   "Deploy tekton TaskRun object",
		Run: func(cmd *cobra.Command, args []string) {
			tr.Namespace = client.Namespace
			tr.Registry = client.Registry
			tr.Wait = client.Wait
			_, err := tr.Deploy("", clientset)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	deployTaskRunCmd.Flags().StringVarP(&tr.Task.Name, "task", "t", "", "Name of task to run")
	deployTaskRunCmd.Flags().StringVarP(&tr.PipelineResource.Name, "resources", "r", "", "Name of pipelineresource to pass into task")
	deployTaskRunCmd.Flags().StringVarP(&tr.RegistrySecret, "secret", "s", "", "Secret name with registry credentials")
	return deployTaskRunCmd
}

func cmdDeployPipelineResource(clientset *client.ConfigSet) *cobra.Command {
	deployPipelineResourceCmd := &cobra.Command{
		Use:     "pipelineresource",
		Aliases: []string{"pipelineresources"},
		Args:    cobra.ExactArgs(1),
		Short:   "Deploy tekton PipelineResource object",
		Run: func(cmd *cobra.Command, args []string) {
			plr.Name = args[0]
			plr.Namespace = client.Namespace
			if _, err := plr.Deploy(clientset); err != nil {
				log.Fatal(err)
			}
			fmt.Println("PipelineResource created")
		},
	}
	deployPipelineResourceCmd.Flags().StringVar(&plr.Source.URL, "url", "", "Git URL to get sources from")
	deployPipelineResourceCmd.Flags().StringVar(&plr.Source.Revision, "rev", "", "Git revision")
	return deployPipelineResourceCmd
}
