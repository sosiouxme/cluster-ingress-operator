package manifests

import (
	"fmt"
	"testing"
	"time"

	ingressv1alpha1 "github.com/openshift/cluster-ingress-operator/pkg/apis/ingress/v1alpha1"
	"github.com/openshift/cluster-ingress-operator/pkg/operator"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestManifests(t *testing.T) {
	config := operator.Config{RouterImage: "quay.io/openshift/router:latest"}
	f := NewFactory(config)

	ci := &ingressv1alpha1.ClusterIngress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: ingressv1alpha1.ClusterIngressSpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"foo": "bar",
				},
			},
			RouteSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"baz": "quux",
				},
			},
		},
	}

	if _, err := f.RouterNamespace(); err != nil {
		t.Errorf("invalid RouterNamespace: %v", err)
	}

	if _, err := f.RouterServiceAccount(); err != nil {
		t.Errorf("invalid RouterServiceAccount: %v", err)
	}

	if _, err := f.RouterClusterRole(); err != nil {
		t.Errorf("invalid RouterClusterRole: %v", err)
	}

	if _, err := f.RouterClusterRoleBinding(); err != nil {
		t.Errorf("invalid RouterClusterRoleBinding: %v", err)
	}

	ds, err := f.RouterDaemonSet(ci)
	if err != nil {
		t.Errorf("invalid RouterDaemonSet: %v", err)
	}

	namespaceSelector := ""
	for _, envVar := range ds.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name == "NAMESPACE_LABELS" {
			namespaceSelector = envVar.Value
			break
		}
	}
	if namespaceSelector == "" {
		t.Error("RouterDaemonSet has no namespace selector")
	} else if namespaceSelector != "foo=bar" {
		t.Errorf("RouterDaemonSet has unexpected namespace selectors: %v",
			namespaceSelector)
	}

	routeSelector := ""
	for _, envVar := range ds.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name == "ROUTE_LABELS" {
			routeSelector = envVar.Value
			break
		}
	}
	if routeSelector == "" {
		t.Error("RouterDaemonSet has no route selector")
	} else if routeSelector != "baz=quux" {
		t.Errorf("RouterDaemonSet has unexpected route selectors: %v",
			routeSelector)
	}

	if len(ds.Spec.Template.Spec.NodeSelector) == 0 {
		t.Error("RouterDaemonSet has no default node selector")
	}

	if ds.Spec.Template.Spec.Volumes[0].Secret == nil {
		t.Error("RouterDaemonSet has no secret volume")
	}

	defaultSecretName := fmt.Sprintf("router-certs-%s", ci.Name)
	if ds.Spec.Template.Spec.Volumes[0].Secret.SecretName != defaultSecretName {
		t.Errorf("RouterDaemonSet expected volume with secret %s, got %s",
			defaultSecretName, ds.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	}

	if svc, err := f.RouterServiceInternal(ci); err != nil {
		t.Errorf("invalid RouterServiceInternal: %v", err)
	} else if svc.Annotations[ServingCertSecretAnnotation] != defaultSecretName {
		t.Errorf("RouterServiceInternal expected serving secret annotation %s, got %s",
			defaultSecretName, svc.Annotations[ServingCertSecretAnnotation])
	}

	secretName := fmt.Sprintf("secret-%v", time.Now().UnixNano())
	ci.Spec.DefaultCertificateSecret = &secretName
	ci.Spec.NodePlacement = &ingressv1alpha1.NodePlacement{
		NodeSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"xyzzy": "quux",
			},
		},
	}
	ds, err = f.RouterDaemonSet(ci)
	if err != nil {
		t.Errorf("invalid RouterDaemonSet: %v", err)
	}
	if len(ds.Spec.Template.Spec.NodeSelector) != 1 ||
		ds.Spec.Template.Spec.NodeSelector["xyzzy"] != "quux" {
		t.Errorf("RouterDaemonSet has unexpected node selector: %#v",
			ds.Spec.Template.Spec.NodeSelector)
	}
	if e, a := config.RouterImage, ds.Spec.Template.Spec.Containers[0].Image; e != a {
		t.Errorf("expect router daemonset %q, got %q", e, a)
	}

	if ds.Spec.Template.Spec.Volumes[0].Secret == nil {
		t.Error("RouterDaemonSet has no secret volume")
	}
	if ds.Spec.Template.Spec.Volumes[0].Secret.SecretName != secretName {
		t.Errorf("RouterDaemonSet expected volume with secret %s, got %s",
			secretName, ds.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	}

	if svc, err := f.RouterServiceInternal(ci); err != nil {
		t.Errorf("invalid RouterServiceInternal: %v", err)
	} else if svc.Annotations[ServingCertSecretAnnotation] != defaultSecretName {
		t.Errorf("RouterServiceInternal expected serving secret annotation %s, got %s",
			defaultSecretName, svc.Annotations[ServingCertSecretAnnotation])
	}

	if _, err := f.RouterServiceCloud(ci); err != nil {
		t.Errorf("invalid RouterServiceCloud: %v", err)
	}
}

func TestDefaultClusterIngress(t *testing.T) {
	ingressDomain := "user.cluster.openshift.com"
	def, err := NewFactory(operator.Config{
		RouterImage:          "test",
		DefaultIngressDomain: ingressDomain,
	}).DefaultClusterIngress()
	if err != nil {
		t.Fatal(err)
	}
	if e, a := ingressDomain, *def.Spec.IngressDomain; e != a {
		t.Errorf("expected default clusteringress ingressDomain=%s, got %s", e, a)
	}
}
