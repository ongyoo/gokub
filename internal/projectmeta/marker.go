package projectmeta

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ongyoo/gokub/internal/manifest"
)

const MarkerFile = "gokub.init"

// WriteMarker records that GOKUB tooling and agent context are available in a
// project. The file is intentionally small and stable so humans and agents can
// inspect it without a dedicated parser.
func WriteMarker(root, version string, m manifest.Manifest) error {
	content := fmt.Sprintf(`# This project is initialized for GOKUB.
schema_version: 1
cli_version: %s
manifest: %s
project: %s
module: %s
agent_skills: enabled
mcp: enabled
`, version, manifest.FileName, m.Name, m.Module)
	return os.WriteFile(filepath.Join(root, MarkerFile), []byte(content), 0o644)
}
