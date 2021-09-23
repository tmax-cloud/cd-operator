package manifestmanager

import (
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("manifest-manager")

type ManifestManager struct {
}

func (m *ManifestManager) GetManifest(info *ApplicationInfo) {

}
func (m *ManifestManager) ApplyManifest(path ManifestPath) {

}
