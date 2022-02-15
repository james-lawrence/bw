package main

type cmdWorkspace struct {
	Create cmdWorkspaceCreate `cmd:"" help:"initialize a workspace"`
}

type cmdWorkspaceCreate struct {
	Directory string `arg:"" help:"path of the workspace directory to create" default:"${vars.bw.default.deployspace.directory}"`
	Example   bool   `help:"include examples" default:"false"`
}
