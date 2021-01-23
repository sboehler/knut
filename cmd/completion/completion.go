// Copyright 2020 Silvio Böhler
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

package completion

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func CreateCmd(rootCmd *cobra.Command) *cobra.Command {
	c := &cobra.Command{
		Use:   "completion [bash|zsh]",
		Short: "output shell completion code [bash|zsh]",
		Long: `To load completions:

Bash:

$ source <(knut completion bash)

# To load completions for each session, execute once:
Linux:
  $ knut completion bash > /etc/bash_completion.d/knut
MacOS:
  $ knut completion bash > /usr/local/etc/bash_completion.d/knut

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions in your current shell session:
$ source <(knut completion zsh)

# To load completions for each session, execute once:
$ knut completion zsh > "${fpath[1]}/_knut"

# You will need to start a new shell for this setup to take effect.
`,

		Args: cobra.ExactValidArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case `bash`:
				rootCmd.GenBashCompletion(os.Stdout)
			case `zsh`:
				rootCmd.GenZshCompletion(os.Stdout)
				io.WriteString(os.Stdout, "\ncompdef _knut knut\n")
			default:
				fmt.Printf("Unknown shell: %s", args[0])
			}
		},
	}

	return c
}
