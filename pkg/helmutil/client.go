package helmutil

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdateObject(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	existingObj := &unstructured.Unstructured{}
	existingObj.SetGroupVersionKind(obj.GroupVersionKind())
	err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, existingObj)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// Not found, create
		return k8sClient.Create(ctx, obj)
	}
	// Found, update
	obj.SetResourceVersion(existingObj.GetResourceVersion())
	return k8sClient.Update(ctx, obj)
}
