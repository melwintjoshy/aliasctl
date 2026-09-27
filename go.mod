module github.com/melwintjoshy/aliasctl

go 1.27.1

// pre-launch builds; v1.0.0 is the first public release
retract [v0.1.0, v0.2.0]

require (
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.9
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/inconshreveable/mousetrap v1.1.0 // indirect
