package cmdopts

import (
	"net"
	"path/filepath"
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
)

// Vars are the kong.Vars used to interpolate the struct tags of the bw commands. defaults provides the
// default agent bind address.
func Vars(defaults agent.Config) kong.Vars {
	return kong.Vars{
		"vars_bw_default_env_name":                         bw.DefaultEnvironmentName,
		"vars_bw_default_deployspace_directory":            bw.DefaultDeployspaceDir,
		"vars_bw_default_deployspace_config_directory":     bw.DefaultDeployspaceConfigDir,
		"vars_bw_default_agent_configuration_location":     bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), ""),
		"vars_bw_default_agent_address":                    defaults.P2PBind.String(),
		"env_bw_agent_bind_primary":                        bw.EnvAgentP2PBind,
		"env_bw_agent_bind_advertised":                     bw.EnvAgentP2PAdvertised,
		"env_bw_agent_bind_secondary":                      bw.EnvAgentP2PAlternatesBind,
		"env_bw_agent_bootstrap_static":                    bw.EnvAgentClusterBootstrap,
		"env_bw_agent_bootstrap_p2p_enabled":               bw.EnvAgentClusterEnableP2P,
		"env_bw_agent_bootstrap_dns_enabled":               bw.EnvAgentClusterEnableDNS,
		"env_bw_agent_bootstrap_aws_autoscaling_enabled":   bw.EnvAgentClusterEnableAWSAutoscaling,
		"env_bw_agent_bootstrap_gcloud_taget_pool_enabled": bw.EnvAgentClusterEnableGoogleCloudPool,
	}
}

// Mappers are the kong type mappers required by the bw commands.
func Mappers() kong.Option {
	mappers := []kong.Option{
		kong.TypeMapper(reflect.TypeFor[*net.IP](), kong.MapperFunc(ParseIP)),
		kong.TypeMapper(reflect.TypeFor[*net.TCPAddr](), kong.MapperFunc(ParseTCPAddr)),
		kong.TypeMapper(reflect.TypeFor[[]*net.TCPAddr](), kong.MapperFunc(ParseTCPAddrArray)),
	}

	return kong.OptionFunc(func(k *kong.Kong) error {
		for _, m := range mappers {
			if err := m.Apply(k); err != nil {
				return err
			}
		}

		return nil
	})
}
