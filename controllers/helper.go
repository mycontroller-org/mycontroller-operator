package controllers

import (
	"os"

	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
)

var openshiftPlatform = false

func isOpenshift() bool {
	return openshiftPlatform
}

func updatePlatform(mgr ctrl.Manager) error {
	log := mgr.GetLogger()
	// update platform
	dClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		log.Error(err, "problem getting platform details")
		return err
	}

	apiList, err := dClient.ServerGroups()
	if err != nil {
		log.Error(err, "problem getting platform details")
		os.Exit(1)
	}

	apiGroups := apiList.Groups
	for _, group := range apiGroups {
		if group.Name == "route.openshift.io" {
			openshiftPlatform = true
		}
	}
	log.Info("platform detail", "is_openshift", openshiftPlatform)
	return nil
}
