package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ReadSnapshotArtifact(id, dest string) (string, error) {
	return readSnapshotArtifact(stateDir, id, dest)
}

func readSnapshotArtifact(root, id, dest string) (string, error) {
	if !validSnapshotID(id) {
		return "", fmt.Errorf("invalid snapshot id")
	}

	data, err := os.ReadFile(filepath.Join(root, "snapshots", id+".json"))
	if err != nil {
		return "", err
	}

	var saved snapshot
	if err := json.Unmarshal(data, &saved); err != nil {
		return "", err
	}

	content, ok := saved.Files[dest]
	if !ok {
		return "", fmt.Errorf("artifact is not in snapshot")
	}

	return content, nil
}

func validSnapshotID(id string) bool {
	return id != "" && !strings.ContainsAny(id, "/\\") && !strings.Contains(id, "..")
}
