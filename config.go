package bw

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/james-lawrence/bw/internal/x/systemx"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

const (
	// DefaultDir agent and client configuration directory, relative to a absolute path.
	DefaultDir = "bearded-wookie"
	// DefaultAgentConfig default filename of the agent configuration.
	DefaultAgentConfig = "agent.config"
	// DefaultDeployspaceDir default directory of the workspace.
	DefaultDeployspaceDir = ".bw"
	// DefaultDeployspaceConfigDir default configuration directory of the workspace.
	DefaultDeployspaceConfigDir = ".bwconfig"
	// DefaultEnvironmentName name of the environment to default to.
	DefaultEnvironmentName = "default"
)

var fallbackUser = user.User{
	Gid:     "0",
	Uid:     "0",
	HomeDir: "/root",
}

// LocateDeployspace - looks for the provided filename up the file tree.
// and returns the path once found, if no path is found then it returns
// the name without a directory, which makes its a relative path.
func LocateDeployspace(name string) string {
	// fallback to root so it'll stop immediately.
	for dir := systemx.WorkingDirectoryOrDefault("/"); dir != "/"; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return name
}

// DefaultConfigFile returns the default configuration file location.
func DefaultConfigFile() string {
	return DefaultLocation(filepath.Join(DefaultEnvironmentName, DefaultAgentConfig), "")
}

// DefaultLocation returns the location of file path to be read based using
// the given name and potentially an override path.
// File locations are checked in the following order:
// {override}/{name}
// ${XDG_CONFIG_HOME}/{configurationDirDefault}/{name}
// ${HOME}/.config/{configurationDirDefault}/{name}
// /etc/{configurationDirDefault}/{name}
//
// if none of the files are found then the last location checked is returned.
func DefaultLocation(name, override string) string {
	user := systemx.CurrentUserOrDefault(fallbackUser)

	envconfig := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), DefaultDir)
	home := filepath.Join(user.HomeDir, ".config", DefaultDir)
	system := filepath.Join("/etc", DefaultDir)

	return locateFile(name, override, envconfig, home, system)
}

// LocateFirstInDir locates the first file in the given directory by name.
func LocateFirstInDir(dir string, names ...string) (result string) {
	for _, name := range names {
		result = filepath.Join(dir, name)
		if _, err := os.Stat(result); err == nil {
			break
		}
	}

	return result
}

// DefaultUserDirLocation returns the user directory location.
func DefaultUserDirLocation(name, override string) string {
	user := systemx.CurrentUserOrDefault(fallbackUser)

	envconfig := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), DefaultDir)
	home := filepath.Join(user.HomeDir, ".config", DefaultDir)

	return DefaultDirectory(name, override, envconfig, home)
}

// DefaultDirLocation looks for a directory one of the default directory locations.
func DefaultDirLocation(rel string) string {
	user := systemx.CurrentUserOrDefault(fallbackUser)

	env := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), DefaultDir)
	home := filepath.Join(user.HomeDir, ".config", DefaultDir)
	system := filepath.Join("/etc", DefaultDir)

	return DefaultDirectory(rel, env, home, system)
}

// DefaultDirectory finds the first directory root that exists and then returns
// that root directory joined with the relative path provided.
func DefaultDirectory(rel string, roots ...string) (path string) {
	for _, root := range roots {
		path = filepath.Join(root, rel)
		if _, err := os.Stat(root); err == nil {
			return path
		}
	}

	return path
}

func locateFile(name string, searchDirs ...string) (result string) {
	for _, dir := range searchDirs {
		result = filepath.Join(dir, name)
		if _, err := os.Stat(result); err == nil {
			break
		}
	}
	return result
}

// ExpandAndDecodeFile ...
func ExpandAndDecodeFile(path string, dst interface{}) (err error) {
	var (
		raw []byte
	)

	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	if raw, err = ioutil.ReadFile(path); err != nil {
		return errors.WithStack(err)
	}

	log.Println("loaded configuration", path)

	return ExpandAndDecode(raw, dst)
}

// ExpandAndDecode expands environment variables within the file at the specified
// path and then decodes it as yaml.
func ExpandAndDecode(raw []byte, dst interface{}) (err error) {
	return ExpandEnvironAndDecode(raw, dst, os.Getenv)
}

// ExpandEnvironAndDecode ...
func ExpandEnvironAndDecode(raw []byte, dst interface{}, mapping func(string) string) (err error) {
	return yaml.Unmarshal([]byte(os.Expand(string(raw), mapping)), dst)
}

// InitializeDeploymentDirectory initializes the directory for the deployments.
func InitializeDeploymentDirectory(root string) (err error) {
	log.Println("creating deploys directory", filepath.Join(root, DirDeploys))
	if err = os.MkdirAll(filepath.Join(root, DirDeploys), 0755); err != nil {
		return errors.WithStack(err)
	}

	if err = os.MkdirAll(filepath.Join(root, DirObservers), 0700); err != nil {
		return errors.WithStack(err)
	}

	// TODO when we start persisting raft state to disk.
	// log.Println("creating raft directory", filepath.Join(root, DirRaft))
	// if err = os.MkdirAll(filepath.Join(root, DirRaft), 0755); err != nil {
	// 	return errors.WithStack(err)
	// }

	// TODO someday when we have plugins.
	// log.Println("creating plugins directory", filepath.Join(root, DirPlugins))
	// if err = os.MkdirAll(filepath.Join(root, DirPlugins), 0755); err != nil {
	// 	return errors.WithStack(err)
	// }

	return nil
}
