// Copyright 2024 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"ctx.sh/coral/pkg/build"
	"ctx.sh/coral/pkg/cmd/coral/agent"
	"ctx.sh/coral/pkg/cmd/coral/controller"
	"github.com/spf13/cobra"
)

const (
	RootUsage     = "coral [COMMAND] [ARG...]"
	RootShortDesc = "Build controller and image sync tool for kubernetes"
	RootLongDesc  = `Coral is a build controller and image sync tool for kubernetes.  It
provides components for watching source repositories for changes and building containers
when changes and conditions are detected.  It also provides a tool for syncrhonizing the
new images to nodes in a cluster based off of node labels bypassing the need for external
registries.`
	ControllerUsage     = "controller [ARG...]"
	ControllerShortDesc = "Start the coral controller"
	ControllerLongDesc  = `Starts the coral controller providing management of the
kubernetes resources and services.`
	AgentUsage     = "agent [ARG...]"
	AgentShortDesc = "Start the coral agent"
	AgentLongDesc  = `Starts the coral agent which ensures the the node contains the configured resources.`
)

type Root struct{}

func NewRoot() *Root {
	return &Root{}
}

func (r *Root) Execute() error {
	if err := r.Command().Execute(); err != nil {
		return err
	}

	return nil
}

func (r *Root) Command() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     RootUsage,
		Short:   RootShortDesc,
		Long:    RootLongDesc,
		Version: build.Version,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	rootCmd.AddCommand(ControllerCommand())
	rootCmd.AddCommand(AgentCommand())
	return rootCmd
}

func ControllerCommand() *cobra.Command {
	c := controller.Controller{}

	cmd := &cobra.Command{
		Use:   ControllerUsage,
		Short: ControllerShortDesc,
		Long:  ControllerLongDesc,
		RunE:  c.RunE,
	}

	cmd.PersistentFlags().StringVarP(&c.Certs, "certs", "c", DefaultCertDir, "specify the webhooks certs directory")
	cmd.PersistentFlags().BoolVarP(&c.LeaderElection, "enable-leader-election", "", DefaultEnableLeaderElection, "enable leader election")
	cmd.PersistentFlags().BoolVarP(&c.SkipInsecureVerify, "skip-insecure-verify", "", DefaultSkipInsecureVerify, "skip certificate verification for the webhooks")
	cmd.PersistentFlags().Int8VarP(&c.LogLevel, "log-level", "", DefaultLogLevel, "set the log level (integer value)")
	cmd.PersistentFlags().StringVarP(&c.Namespace, "namespace", "n", DefaultNamespace, "limit the coral scope to a specific namespace")
	return cmd
}

func AgentCommand() *cobra.Command {
	a := agent.Agent{}
	cmd := &cobra.Command{
		Use:   AgentUsage,
		Short: AgentShortDesc,
		Long:  AgentLongDesc,
		RunE:  a.RunE,
	}

	cmd.PersistentFlags().Int8VarP(&a.LogLevel, "log-level", "v", DefaultLogLevel, "set the log level (integer value)")
	cmd.PersistentFlags().StringVarP(&a.ContainerdAddr, "containerd-addr", "A", DefaultContainerdAddr, "set the containerd address")
	cmd.PersistentFlags().IntVarP(&a.MaxConcurrentReconcilers, "max-concurrent-reconcilers", "", DefaultMaxConcurrentReconcilers, "set the max concurrency for resource reconciliation")
	cmd.PersistentFlags().IntVarP(&a.MaxConcurrentPullers, "max-concurrent-pullers", "", DefaultMaxConcurrentPullers, "set the max concurrency for pulling images")

	return cmd
}
