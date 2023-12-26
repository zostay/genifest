package k8s

import (
	"context"
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/zostay/genifest/pkg/log"
)

const (
	AnnotationIngressService       = "qubling.cloud/ingress-service"
	AnnotationLegacyIngressService = "zostay-ingress-service"

	AnnotationHostAliases       = "qubling.cloud/host-aliases"
	AnnotationLegacyHostAliases = "zostay-host-aliases"

	AnnotationServiceName       = "qubling.cloud/service-name"
	AnnotationLegacyServiceName = "zostay-dns-srv/service-name"

	AnnotationDestination       = "qubling.cloud/destination"
	AnnotationLegacyDestination = "zostay-dns-srv/destination"

	AnnotationDeprecatedDNSNames = "qubling.cloud/deprecated-dns-names"

	LabelQublingEnabled = "qubling.cloud/infra-enabled"
)

// IngressHosts represent a collection of hosts that will need to be configured
// for DNS.
type IngressHosts struct {
	Name          string
	Namespace     string
	Kind          string
	Hosts         []string
	LoadBalancers []string
}

// lookupAnnotation returns the first annotation value matching the given set of
// names.
func lookupAnnotation(obj metav1.ObjectMeta, names ...string) string {
	for _, name := range names {
		if v, ok := obj.Annotations[name]; ok {
			return v
		}
	}

	return ""
}

// getDeprecatedDNSNames returns a map containing names that should be ignored
// for the sake of DNS naming. This was created to simplify a cluster migration.
func (c *Client) getDeprecatedDNSNames(ing metav1.ObjectMeta) map[string]struct{} {
	deprecated := make(map[string]struct{})
	if dn, ok := ing.Annotations[AnnotationDeprecatedDNSNames]; ok {
		depNames := strings.Split(dn, ",")
		for _, d := range depNames {
			k := strings.TrimSpace(d)
			deprecated[k] = struct{}{}
		}
	}

	return deprecated
}

// listNetworkingV1IngressesHosts returns a set of IngressHosts objects for
// those ingresses in the networking.k8s.io/v1/Ingress GVK.
func (c *Client) listNetworkingV1IngressesHosts(
	ctx context.Context,
	ns string,
	ings []IngressHosts,
) ([]IngressHosts, error) {
	v1ings, err := c.kube.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(labels.Set{
			LabelQublingEnabled: "true",
		}),
	})
	if err != nil && !errors.IsNotFound(err) {
		return ings, err
	}

	for _, ing := range v1ings.Items {
		ih := IngressHosts{
			Name:          ing.Name,
			Kind:          "v1/Ingress",
			Namespace:     ing.Namespace,
			Hosts:         []string{},
			LoadBalancers: []string{},
		}

		for _, ingDef := range ing.Status.LoadBalancer.Ingress {
			ih.LoadBalancers = append(ih.LoadBalancers, ingDef.Hostname)
		}

		deprecated := c.getDeprecatedDNSNames(ing.ObjectMeta)

		for _, rule := range ing.Spec.Rules {
			if rule.Host == "" {
				continue
			}

			if _, skip := deprecated[rule.Host]; skip {
				continue
			}

			ih.Hosts = append(ih.Hosts, rule.Host)
		}

		ings = append(ings, ih)
	}

	return ings, nil
}

// listNetworkingV1betaIngresses returns the IngressHosts objects for all the
// networking.k8s.io/v1beta1/Ingress GVK objects.
func (c *Client) listNetworkingV1beta1IngressesHosts(
	ctx context.Context,
	ns string,
	ings []IngressHosts,
) ([]IngressHosts, error) {
	v1bings, err := c.kube.NetworkingV1beta1().Ingresses(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(labels.Set{
			LabelQublingEnabled: "true",
		}),
	})
	if err != nil && !errors.IsNotFound(err) {
		return ings, err
	}

	for _, ing := range v1bings.Items {
		ih := IngressHosts{
			Name:          ing.Name,
			Kind:          "v1beta1/Ingress",
			Namespace:     ing.Namespace,
			Hosts:         []string{},
			LoadBalancers: []string{},
		}

		for _, ingDef := range ing.Status.LoadBalancer.Ingress {
			ih.LoadBalancers = append(ih.LoadBalancers, ingDef.Hostname)
		}

		deprecated := c.getDeprecatedDNSNames(ing.ObjectMeta)

		for _, rule := range ing.Spec.Rules {
			if rule.Host == "" {
				continue
			}

			if _, skip := deprecated[rule.Host]; skip {
				continue
			}

			ih.Hosts = append(ih.Hosts, rule.Host)
		}

		ings = append(ings, ih)
	}

	return ings, nil
}

// listSyntheticIngressHosts returns all the IngressHosts objects representing
// DNS names discovered via synthetic TCP/UDP ingresses stored in marked
// ConfigMaps.
func (c *Client) listSyntheticIngressesHosts(
	ctx context.Context,
	ns string,
	ings []IngressHosts,
) ([]IngressHosts, error) {
	cms, err := c.kube.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(labels.Set{
			"for":               "ingress",
			LabelQublingEnabled: "true",
		}),
	})
	if err != nil && !errors.IsNotFound(err) {
		return ings, err
	}

	for _, cm := range cms.Items {
		ingSvcFullName := lookupAnnotation(
			cm.ObjectMeta,
			AnnotationIngressService,
			AnnotationLegacyIngressService,
		)
		if ingSvcFullName == "" {
			return ings, fmt.Errorf("please set missing %q annotation on ConfigMap %q in namespace %q", AnnotationIngressService, ns, cm.GetName())
		}

		var ingNamespace, ingSvcName string
		if strings.ContainsRune(ingSvcFullName, '/') {
			ps := strings.SplitN(ingSvcFullName, "/", 2)
			ingNamespace, ingSvcName = ps[0], ps[1]
		} else {
			ingNamespace = "default"
			ingSvcName = ingSvcFullName
		}

		ingSvc, err := c.kube.CoreV1().Services(ingNamespace).Get(ctx, ingSvcName, metav1.GetOptions{})
		if err != nil {
			return ings, err
		}

		ih := IngressHosts{
			Name:          ingSvcName,
			Namespace:     ingNamespace,
			Kind:          "v1/ConfigMap[synthetic-ingress]",
			Hosts:         []string{},
			LoadBalancers: []string{},
		}

		for _, ingDef := range ingSvc.Status.LoadBalancer.Ingress {
			ih.LoadBalancers = append(ih.LoadBalancers, ingDef.Hostname)
		}

		// TODO use _ here as lbPort?
		for _, svcLookup := range cm.Data {
			fixSvcLookup := strings.SplitN(svcLookup, ":", 3)
			if len(fixSvcLookup) > 2 {
				svcLookup = strings.Join(fixSvcLookup[:2], ":")
			}
			fullSvcName, _, err := net.SplitHostPort(svcLookup)
			if err != nil {
				return ings, err
			}

			ps := strings.SplitN(fullSvcName, "/", 2)
			localSvcNamespace, localSvcName := ps[0], ps[1]

			svc, err := c.kube.CoreV1().Services(localSvcNamespace).Get(ctx, localSvcName, metav1.GetOptions{})
			if err != nil {
				return ings, err
			}

			hostAliases := lookupAnnotation(
				svc.ObjectMeta,
				AnnotationHostAliases,
				AnnotationLegacyHostAliases,
			)

			hostnames := strings.Split(hostAliases, ",")
			ih.Hosts = append(ih.Hosts, hostnames...)
		}

		ings = append(ings, ih)
	}

	return ings, nil
}

// ListIngressesHosts returns a list of all IngressHosts objects for every kind of
// supported ingress. This includes Ingress objects as well as synthetic ingress
// in ConfigMaps marked for=ingress.
func (c *Client) ListIngressesHosts(
	ctx context.Context,
	ns string,
) ([]IngressHosts, error) {
	ings := make([]IngressHosts, 0)

	ings, err := c.listNetworkingV1IngressesHosts(ctx, ns, ings)
	if err != nil {
		return nil, err
	}

	ings, err = c.listNetworkingV1beta1IngressesHosts(ctx, ns, ings)
	if err != nil {
		return nil, err
	}

	ings, err = c.listSyntheticIngressesHosts(ctx, ns, ings)
	if err != nil {
		return nil, err
	}

	return ings, nil
}

type ServiceDnsInfo struct {
	ServiceName string
	Destination string
	Port        int
}

// ListServiceDnsInfo returns all the DNS information for services in the given
// namespace. This uses the AnnotationServiceName and
// AnnotationLegacyServiceName as well as the AnnotationDestination and
// AnnotationLegacyDestination annotations to find the information and return
// it.
func (c *Client) ListServiceDnsInfo(ctx context.Context, ns string) ([]ServiceDnsInfo, error) {
	svcs, err := c.kube.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	services := make([]ServiceDnsInfo, 0, len(svcs.Items))
	for _, svc := range svcs.Items {
		srv := lookupAnnotation(
			svc.ObjectMeta,
			AnnotationServiceName,
			AnnotationLegacyServiceName,
		)
		if srv == "" {
			continue
		}

		dst := lookupAnnotation(
			svc.ObjectMeta,
			AnnotationDestination,
			AnnotationLegacyDestination,
		)
		if dst == "" {
			log.LineAndSayf("WARN", "service %q in namespace %q found with %q annotation, but not %q annotation",
				svc.GetName(),
				svc.GetNamespace(),
				AnnotationServiceName,
				AnnotationDestination,
			)
			continue
		}

		if svc.Spec.Type != corev1.ServiceTypeNodePort {
			log.LineAndSayf("WARN", "service %q in namespace %q found with %q annotation and %q annotation, but without NodePort",
				svc.GetName(),
				svc.GetNamespace(),
				AnnotationServiceName,
				AnnotationDestination,
			)
			continue
		}

		log.Linef("PROCESS", "Processing DNS SRV record for %q", svc.Name)

		var port int
		for _, svcPort := range svc.Spec.Ports {
			if svcPort.NodePort > 0 {
				port = int(svcPort.NodePort)
				break
			}
		}

		services = append(services, ServiceDnsInfo{
			ServiceName: srv,
			Destination: dst,
			Port:        port,
		})
	}

	return services, nil
}
