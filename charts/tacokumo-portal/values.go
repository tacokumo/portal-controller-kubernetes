package tacokumoportal

// Valuesに型を付けるための構造体
// ユニットテストでvalues.yamlがこの構造に従っていることを自動検査します。
type Values struct {
	UI         UIValues `yaml:"ui"`
	Namespace  string   `yaml:"namespace"`
	NamePrefix string   `yaml:"namePrefix"`
}

type UIValues struct {
	Service          ServiceValues     `yaml:"service"`
	ReplicaCount     int               `yaml:"replicaCount"`
	Annotations      map[string]string `yaml:"annotations"`
	PodAnnotations   map[string]string `yaml:"podAnnotations"`
	Image            string            `yaml:"image"`
	ImagePullSecrets []string          `yaml:"imagePullSecrets"`
	ImagePullPolicy  string            `yaml:"imagePullPolicy"`
}

type ServiceValues struct {
	Port       int `yaml:"port"`
	TargetPort int `yaml:"targetPort"`
}
