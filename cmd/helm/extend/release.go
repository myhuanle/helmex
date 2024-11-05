package extend

import "fmt"

func releaseName(k8s, namespace, serviceName string) string {
	v := fmt.Sprintf("%s-%s-%s", serviceName, k8s, namespace)
	if len(v) > 64 {
		panic(fmt.Sprintf("release name %s is too long, it exceeded 64", v))
	}
	return v
}
