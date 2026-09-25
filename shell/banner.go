package shell

import (
	"fmt"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

const banner = `
   █████████   ████   ███                      █████████   █████    ████ 
  ███░░░░░███ ░░███  ░░░                      ███░░░░░███ ░░███    ░░███ 
 ░███    ░███  ░███  ████   ██████    █████  ███     ░░░  ███████   ░███ 
 ░███████████  ░███ ░░███  ░░░░░███  ███░░  ░███         ░░░███░    ░███ 
 ░███░░░░░███  ░███  ░███   ███████ ░░█████ ░███           ░███     ░███ 
 ░███    ░███  ░███  ░███  ███░░███  ░░░░███░░███     ███  ░███ ███ ░███ 
 █████   █████ █████ █████░░████████ ██████  ░░█████████   ░░█████  █████
░░░░░   ░░░░░ ░░░░░ ░░░░░  ░░░░░░░░ ░░░░░░    ░░░░░░░░░     ░░░░░  ░░░░░ 
`

func RenderBanner(env *resolver.Environment, configPath string) string {
	var output strings.Builder

	for _, line := range strings.Split(strings.TrimRight(banner, "\n"), "\n") {
		fmt.Fprintf(&output, "printf '%%s\\n' %s\n", shellQuote(line))
	}

	output.WriteString("printf '\\n'\n")

	fmt.Fprintf(&output, "printf '%%s\\n' %s\n",
		shellQuote("  Environment: "+env.Name),
	)

	fmt.Fprintf(&output, "printf '%%s\\n' %s\n",
		shellQuote("  Config:      "+configPath),
	)

	output.WriteString("printf '\\n'\n")

	return output.String()
}
