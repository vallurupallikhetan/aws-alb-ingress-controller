package lb

import (
	"os"
	"testing"

	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/controller/dummy"

	api "k8s.io/api/core/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/controller/store"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/pkg/util/log"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/pkg/util/types"
)

const (
	clusterName = "cluster1"
	ingressName = "ingress1"
	sg1         = "sg-123"
	sg2         = "sg-abc"
	tag1Key     = "tag1"
	tag1Value   = "value1"
	tag2Key     = "tag2"
	tag2Value   = "value2"
)

func TestNewDesiredLoadBalancer(t *testing.T) {
	dummyStore := store.NewDummy()
	ing := dummy.NewIngress()

	cfg := dummyStore.GetConfig()
	cfg.ALBNamePrefix = clusterName
	dummyStore.SetConfig(cfg)

	ia := dummyStore.GetIngressAnnotationsResponse
	ia.LoadBalancer.Scheme = aws.String(elbv2.LoadBalancerSchemeEnumInternal)
	ia.LoadBalancer.SecurityGroups = types.AWSStringSlice{aws.String(sg1), aws.String(sg2)}
	ia.LoadBalancer.WebACLId = aws.String("web acl id")

	commonTags := types.ELBv2Tags{
		{
			Key:   aws.String(tag1Key),
			Value: aws.String(tag1Value),
		},
		{
			Key:   aws.String(tag2Key),
			Value: aws.String(tag2Value),
		},
	}

	lbOpts := &NewDesiredLoadBalancerOptions{
		ExistingLoadBalancer: nil,
		Ingress:              ing,
		Logger:               log.New("test"),
		CommonTags:           commonTags,
		Store:                dummyStore,
	}

	os.Setenv("AWS_VPC_ID", "vpc-id")
	expectedID := createLBName(api.NamespaceDefault, ingressName, clusterName)
	l, err := NewDesiredLoadBalancer(lbOpts)
	if err != nil {
		t.Error(err.Error())
	}

	key1, _ := l.tags.desired.Get(tag1Key)
	switch {
	case *l.lb.desired.LoadBalancerName != expectedID:
		t.Errorf("LB ID was wrong. Expected: %s | Actual: %s", expectedID, l.id)
	case *l.lb.desired.Scheme != *ia.LoadBalancer.Scheme:
		t.Errorf("LB scheme was wrong. Expected: %s | Actual: %s", *ia.LoadBalancer.Scheme, *l.lb.desired.Scheme)
	case *l.lb.desired.SecurityGroups[0] == sg2: // note sgs are sorted during checking for modification needs.
		t.Errorf("Security group was wrong. Expected: %s | Actual: %s", sg2, *l.lb.desired.SecurityGroups[0])
	case key1 != tag1Value:
		t.Errorf("Tag was invalid. Expected: %s | Actual: %s", tag1Value, key1)
	case *l.options.desired.webACLId != *ia.LoadBalancer.WebACLId:
		t.Errorf("Web ACL ID was invalid. Expected: %s | Actual: %s", *ia.LoadBalancer.WebACLId, *l.options.desired.webACLId)

	}
}

// Temporarily disabled until we mock out the AWS API calls involved
// func TestNewCurrentLoadBalancer(t *testing.T) {
// 	l, err := NewCurrentLoadBalancer(lbOpts)
// 	if err != nil {
// 		t.Errorf("Failed to create LoadBalancer object from existing elbv2.LoadBalancer."+
// 			"Error: %s", err.Error())
// 		return
// 	}

// 	switch {
// 	case *l.lb.current.LoadBalancerName != expectedName:
// 		t.Errorf("Current LB created returned improper LoadBalancerName. Expected: %s | "+
// 			"Desired: %s", expectedName, *l.lb.current.LoadBalancerName)
// 	case *l.options.current.webACLId != *currentWeb:
// 		t.Errorf("Current LB created returned improper Web ACL Id. Expected: %s | "+
// 			"Desired: %s", *currentWeb, *l.options.current.webACLId)
// 	}
// }

// TestLoadBalancerFailsWithInvalidName ensures an error is returned when the LoadBalancerName does
// match what would have been calculated for the LB from the clustername, ingressname, and
// namespace
