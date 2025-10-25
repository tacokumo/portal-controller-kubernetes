package v1alpha1

const (
	ManagedByLabelKey = "tacokumo.io/managed-by"
)

func IsManagedByTacoKumo(labels map[string]string) bool {
	if val, exists := labels[ManagedByLabelKey]; exists {
		return val == "portal-controller"
	}
	return false
}
