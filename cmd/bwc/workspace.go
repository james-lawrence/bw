package main

type cmdWorkspace struct {
	Create cmdWorkspaceCreate `cmd:"" help:"initialize a workspace"`
}

type cmdWorkspaceCreate struct {
	Directory string `arg:"" help:"path of the workspace directory to create" default:"${vars_bw_default_deployspace_directory}"`
	Example   bool   `help:"include examples" default:"false"`
}
