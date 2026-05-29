package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/shim"
)

// Init initializes the helm-env environment.
// If HELMENV_ROOT is not set, it prints setup instructions.
// If set, it creates the versions directory, generates the shim, and outputs
// shell initialization code.
func Init() error {
	root, ok := config.GetHelmEnvRoot()
	if !ok {
		fmt.Fprintln(os.Stderr, `HELMENV_ROOT not set. Set HELMENV_ROOT for your shell using the below commands.
echo 'export HELMENV_ROOT="$HOME/.helmenv"' >> ~/.bashrc
or
echo 'export HELMENV_ROOT="$HOME/.helmenv"' >> ~/.zshrc`)
		return fmt.Errorf("HELMENV_ROOT not set")
	}

	// Create versions directory
	versionsDir := filepath.Join(root, "versions")
	if err := os.MkdirAll(versionsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	// Generate shim script
	if err := shim.GenerateShimScript(root); err != nil {
		return fmt.Errorf("failed to generate shim: %w", err)
	}

	// Output shell initialization code
	fmt.Print(shim.GenerateShellInit(root))

	return nil
}
