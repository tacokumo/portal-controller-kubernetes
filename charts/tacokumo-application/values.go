package tacokumoapplication

// Valuesに型を付けるための構造体
// ユニットテストでvalues.yamlがこの構造に従っていることを自動検査します。
type Values struct {
	// Mainは、ユーザがデプロイしたいアプリケーションのコンテナについての設定
	Main MainApplicationValues `yaml:"main"`
}

// MainApplicationValuesは、ユーザがデプロイしたいアプリケーションのコンテナについての設定を表します。
type MainApplicationValues struct {
	ApplicationName  string      `yaml:"applicationName"`
	ReplicaCount     int         `yaml:"replicaCount"`
	Image            ImageValues `yaml:"image"`
	ImagePullSecrets []string    `yaml:"imagePullSecrets"`
}

type ImageValues struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	PullPolicy string `yaml:"pullPolicy"`
}
