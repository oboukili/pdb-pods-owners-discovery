package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"

	policy "k8s.io/api/policy/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ParentResource struct {
	Namespace  string
	Name       string
	Kind       string
	APIVersion string
	PDBName    string
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// create the API client
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	listOpts := meta.ListOptions{Limit: 9000}
	podDisruptionBudgets := make([]policy.PodDisruptionBudget, 0)
	activePDBs := make([]policy.PodDisruptionBudget, 0)
	parents := make([]ParentResource, 0)
	// List namespaces
	_namespaces, err := c.CoreV1().Namespaces().List(context.TODO(), listOpts)
	if err != nil {
		panic(fmt.Errorf("error listing namespaces: %s", err.Error()))
	}
	if _namespaces == nil {
		panic(fmt.Errorf("no namespaces found"))
	}
	namespaces := _namespaces.Items
	// Get all PDBs
	for _, n := range namespaces {
		_pdbs, err := c.PolicyV1beta1().PodDisruptionBudgets(n.Name).List(context.TODO(), listOpts)
		if err != nil {
			panic(fmt.Errorf("error listing PodDisruptionBudgets: %s", err.Error()))
		}
		if _pdbs != nil {
			for _, p := range _pdbs.Items {
				podDisruptionBudgets = append(podDisruptionBudgets, p)
			}
		}
	}
	// Get non obsolete podDisruptionBudgets, e.g. active ones
	for _, pdb := range podDisruptionBudgets {
		if pdb.Status.ExpectedPods > 0 && pdb.Status.DisruptionsAllowed == 0 {
			activePDBs = append(activePDBs, pdb)
		}
	}
	// Get all pods that match active PDBs namespaces and labels
	for _, activePDB := range activePDBs {
		pods, err := c.CoreV1().Pods(activePDB.Namespace).List(context.TODO(), listOpts)
		if err != nil {
			panic(fmt.Errorf("error listing pods: %s", err.Error()))
		}
		for _, pod := range pods.Items {
			// Matching all pods that match all PDB label match selector
			matchingLabelsCount := 0
			for pdbk, pdbv := range activePDB.Spec.Selector.MatchLabels {
				for pk, pv := range pod.ObjectMeta.Labels {
					if pdbk == pk && pdbv == pv {
						matchingLabelsCount++
					}
				}
			}
			if matchingLabelsCount == len(activePDB.Spec.Selector.MatchLabels)-1 {
				for _, or := range pod.OwnerReferences {
					switch or.Kind {
					case "ReplicaSet":
						replicaSets, err := c.AppsV1().ReplicaSets(activePDB.Namespace).List(context.TODO(), listOpts)
						if err != nil {
							panic(fmt.Errorf("error listing replica sets: %s", err.Error()))
						}
						for _, rs := range replicaSets.Items {
							for _, rsp := range getRSParents(rs.OwnerReferences, activePDB) {
								parents = append(parents, ParentResource{
									Namespace:  activePDB.Namespace,
									Name:       rsp.Name,
									Kind:       rsp.Kind,
									APIVersion: rsp.APIVersion,
									PDBName:    activePDB.Name,
								})
							}
						}
					default:
						parents = append(parents, ParentResource{
							Namespace:  activePDB.Namespace,
							Name:       or.Name,
							Kind:       or.Kind,
							APIVersion: or.APIVersion,
							PDBName:    activePDB.Name,
						})
					}
				}
			}
		}
	}
	// Remove duplicates
	mapParents := make(map[string]ParentResource, 0)
	for _, p := range parents {
		uid := fmt.Sprintf("%s/%s/%s", p.Namespace, p.Kind, p.Name)
		mapParents[uid] = ParentResource{
			Namespace:  p.Namespace,
			Name:       p.Name,
			Kind:       p.Kind,
			APIVersion: p.APIVersion,
			PDBName:    p.PDBName,
		}
	}
	// Pretty print results
	t := table.NewWriter()
	t.SetTitle("Kubernetes resources impacted by active PodDisruptionBudgets")
	t.SetOutputMirror(os.Stdout)
	t.SortBy([]table.SortBy{{Name: "PDB", Mode: table.Asc}})
	t.AppendHeader(table.Row{"Name", "Namespace", "Group Version Kind", "PDB"})
	for _, parent := range mapParents {
		t.AppendRow([]interface{}{
			parent.Name,
			parent.Namespace,
			fmt.Sprintf("%s/%s", parent.APIVersion, parent.Kind),
			parent.PDBName,
		})
	}
	t.Render()
}

func getRSParents(ownerReferences []meta.OwnerReference, pdb policy.PodDisruptionBudget) (
	parents []ParentResource) {
	for _, or := range ownerReferences {
		parents = append(parents, ParentResource{
			Namespace:  pdb.Namespace,
			Name:       or.Name,
			Kind:       or.Kind,
			APIVersion: or.APIVersion,
			PDBName:    pdb.Name,
		})
	}
	return
}
