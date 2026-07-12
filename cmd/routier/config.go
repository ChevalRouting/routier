package main

type CLIConfig struct {
	LogLevel   string
	ConfigFile string
}

var cli = &CLIConfig{}

const defaultConfigPath = "/etc/routier/config.yml"

func configArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	return defaultConfigPath
}
