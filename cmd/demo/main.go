package main

import (
	"context"
	"os"
	"strings"
	"time"

	compute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	retryWaitTime  = pflag.Int("retry-wait-time", 20, "retry wait time in seconds")
	resource       = pflag.String("aad-resourcename", "https://management.azure.com/", "resourcename for aad id")
	subscriptionID = pflag.String("subscriptionID", "c1089427-83d3-4286-9f35-5af546a6eb67", "subscriptionID for test")
	clientID       = pflag.String("clientID", "89f69b3d-5b41-4b14-afbf-18fd96104e14", "clientID for the msi id")
	resourceGroup  = pflag.String("resourceGroup", "MC_nbhatia-eu-01_eu-1_eastus", "any resource group with reader permission to the aad object")
)

func main() {
	pflag.Parse()

	podname := os.Getenv("MY_POD_NAME")
	podnamespace := os.Getenv("MY_POD_NAME")
	podip := os.Getenv("MY_POD_IP")

	log.Infof("starting demo pod %s/%s %s", podnamespace, podname, podip)

	logger := log.WithFields(log.Fields{
		"podnamespace": podnamespace,
		"podname":      podname,
		"podip":        podip,
	})

	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		logger.Fatalf("failed to get msiendpoint, %+v", err)
	}

	for {
		doARMOperations(logger, *subscriptionID, *resourceGroup)

		t1 := testMSIEndpoint(logger, msiEndpoint, *resource)
		if t1 == nil {
			logger.Errorf("testMSIEndpoint failed, %+v", err)
			continue
		}

		t2 := testMSIEndpointFromUserAssignedID(logger, msiEndpoint, *clientID, *resource)
		if t2 == nil {
			logger.Errorf("testMSIEndpointFromUserAssignedID failed, %+v", err)
			continue
		}

		if !strings.EqualFold(t1.AccessToken, t2.AccessToken) {
			logger.Errorf("msi, emsi test failed %+v %+v", t1, t2)
		}

		time.Sleep(time.Duration(*retryWaitTime) * time.Second)
	}
}

func doARMOperations(logger *log.Entry, subscriptionID, resourceGroup string) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		logger.Errorf("failed NewAuthorizerFromEnvironment  %+v", authorizer)
		return
	}
	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer
	vmlist, err := vmClient.List(context.Background(), resourceGroup)
	if err != nil {
		logger.Errorf("failed list all vm %+v", err)
		return
	}

	logger.Infof("succesfull doARMOperations vm count %d", len(vmlist.Values()))
}

func testMSIEndpoint(logger *log.Entry, msiEndpoint, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		logger.Errorf("failed to acquire a token using the MSI VM extension, Error: %+v", err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalTokenFromMSI using the MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	logger.Infof("succesfully acquired a token using the MSI, msiEndpoint(%s)", msiEndpoint)
	return &token
}

func testMSIEndpointFromUserAssignedID(logger *log.Entry, msiEndpoint, userAssignedID, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
	if err != nil {
		logger.Errorf("failed NewServicePrincipalTokenFromMSIWithUserAssignedID, clientID: %s Error: %+v", userAssignedID, err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalToken userAssignedID MSI, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
		return nil
	}
	logger.Infof("succesfully acquired a token, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
	return &token
}