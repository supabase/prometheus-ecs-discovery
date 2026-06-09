package main

import (
	"maps"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func fargateTask(ipv4, ipv6 string, extraLabels map[string]string) *AugmentedTask {
	labels := map[string]string{*prometheusPortLabel: "9100"}
	maps.Copy(labels, extraLabels)

	ni := ecstypes.NetworkInterface{}
	if ipv4 != "" {
		ni.PrivateIpv4Address = aws.String(ipv4)
	}
	if ipv6 != "" {
		ni.Ipv6Address = aws.String(ipv6)
	}

	return &AugmentedTask{
		Task: &ecstypes.Task{
			TaskArn:    aws.String("arn:task"),
			Group:      aws.String("group"),
			ClusterArn: aws.String("arn:cluster"),
			LaunchType: ecstypes.LaunchTypeFargate,
			Containers: []ecstypes.Container{{
				Name:              aws.String("app"),
				ContainerArn:      aws.String("arn:container"),
				NetworkInterfaces: []ecstypes.NetworkInterface{ni},
			}},
		},
		TaskDefinition: &ecstypes.TaskDefinition{
			Family:   aws.String("family"),
			Revision: 1,
			ContainerDefinitions: []ecstypes.ContainerDefinition{{
				Name:         aws.String("app"),
				Image:        aws.String("app:latest"),
				DockerLabels: labels,
			}},
		},
	}
}

func TestExporterInformationIPSelection(t *testing.T) {
	const v4, v6 = "10.0.0.1", "2001:db8::1"
	preferV6 := map[string]string{*prometheusPreferIPv6Label: "true"}

	cases := []struct {
		name   string
		ipv4   string
		ipv6   string
		labels map[string]string
		want   string
	}{
		{"v4 only", v4, "", nil, "10.0.0.1:9100"},
		{"v6 only", "", v6, nil, "[2001:db8::1]:9100"},
		{"dual, no label", v4, v6, nil, "10.0.0.1:9100"},
		{"dual, prefer v6", v4, v6, preferV6, "[2001:db8::1]:9100"},
		{"prefer v6 but none, fall back", v4, "", preferV6, "10.0.0.1:9100"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			infos := fargateTask(tc.ipv4, tc.ipv6, tc.labels).ExporterInformation()
			if len(infos) != 1 {
				t.Fatalf("got %d infos, want 1", len(infos))
			}
			if got := infos[0].Targets[0]; got != tc.want {
				t.Errorf("target = %q, want %q", got, tc.want)
			}
		})
	}
}
