package component

import (
	"halkyon.io/api/component/clientset/versioned/typed/component/v1beta1"
	v1beta12 "halkyon.io/api/component/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type client struct {
	client v1beta1.ComponentInterface
	ns     string
}

func (lc client) Create(toCreate runtime.Object) error {
	_, err := lc.client.Create(toCreate.(*v1beta12.Component))
	return err
}

func (lc client) Get(name string, options v1.GetOptions) error {
	_, err := lc.client.Get(name, options)
	return err
}

func (lc client) GetKnown() []string {
	list, err := lc.client.List(v1.ListOptions{})
	if err != nil {
		return []string{}
	}
	items := list.Items
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func (lc client) Delete(name string, options *v1.DeleteOptions) error {
	return lc.client.Delete(name, options)
}

func (lc client) GetNamespace() string {
	return lc.ns
}
