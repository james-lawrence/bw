package cmdopts

type BeardedWookieEnv struct {
	Environment string `arg:"" name:"environment" predictor:"bw.environment" default:"${vars_bw_default_env_name}"`
}

type BeardedWookieEnvRequired struct {
	Environment string `arg:"" name:"environment" predictor:"bw.environment"`
}
