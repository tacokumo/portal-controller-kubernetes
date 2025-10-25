package tacokumoapplication

// Valuesに型を付けるための構造体
// ユニットテストでvalues.yamlがこの構造に従っていることを自動検査します。
type Values struct {
	// Mainは、ユーザがデプロイしたいアプリケーションのコンテナについての設定
	Main MainApplicationValues `yaml:"main"`
}

// MainApplicationValuesは、ユーザがデプロイしたいアプリケーションのコンテナについての設定を表します。
type MainApplicationValues struct {
	ApplicationName  string            `yaml:"applicationName"`
	ReplicaCount     int               `yaml:"replicaCount"`
	Image            string            `yaml:"image"`
	ImagePullPolicy  string            `yaml:"imagePullPolicy"`
	ImagePullSecrets []string          `yaml:"imagePullSecrets"`
	Annotations      map[string]string `yaml:"annotations"`
	PodAnnotations   map[string]string `yaml:"podAnnotations"`
}
