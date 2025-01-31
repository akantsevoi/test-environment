package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func getK8sClient() (*kubernetes.Clientset, error) {
	output, err := exec.Command("kind", "get", "kubeconfig", "--name", "oltp-multi-region").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get kind kubeconfig: %v", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(output)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return clientset, nil
}

func findLeaderNode(clientset *kubernetes.Clientset) (string, string, error) {
	pods, err := clientset.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=maroon",
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to list pods: %v", err)
	}

	var leaderPod string
	for _, pod := range pods.Items {
		logs, err := clientset.CoreV1().Pods("default").GetLogs(pod.Name, &corev1.PodLogOptions{}).Do(context.Background()).Raw()
		if err != nil {
			continue
		}

		lines := strings.Split(string(logs), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.Contains(lines[i], "became leader") {
				parts := strings.Split(lines[i], "Pod ")
				if len(parts) > 1 {
					leaderPod = strings.Split(parts[1], " became")[0]
					break
				}
			}
		}
		if leaderPod != "" {
			break
		}
	}

	if leaderPod == "" {
		return "", "", fmt.Errorf("no leader found")
	}

	pod, err := clientset.CoreV1().Pods("default").Get(context.Background(), leaderPod, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get leader pod: %v", err)
	}

	return leaderPod, pod.Spec.NodeName, nil
}

func getNodeContainer(nodeName string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", fmt.Errorf("failed to create docker client: %v", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %v", err)
	}

	for _, container := range containers {
		fmt.Println("Names: ", container.Names)
		for _, name := range container.Names {
			// Docker container names start with '/'
			if name == "/"+nodeName {
				return container.ID, nil
			}
		}
	}

	return "", fmt.Errorf("container for node %s not found", nodeName)
}

func waitForNodeReady(clientset *kubernetes.Clientset, nodeName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for node to be ready")
		default:
			node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err == nil {
				for _, cond := range node.Status.Conditions {
					if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
						return nil
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func main() {
	clientset, err := getK8sClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	log.Printf("Finding leader node...")
	leaderPod, nodeName, err := findLeaderNode(clientset)
	if err != nil {
		log.Fatalf("Failed to find leader: %v", err)
	}
	log.Printf("Found leader pod %s on node %s", leaderPod, nodeName)

	containerID, err := getNodeContainer(nodeName)
	if err != nil {
		log.Fatalf("Failed to get container ID: %v", err)
	}
	log.Printf("Found container ID: %s", containerID)

	log.Printf("Stopping node %s...", nodeName)
	cmd := exec.Command("docker", "stop", containerID)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to stop node: %v", err)
	}
	log.Printf("Node stopped successfully")

	log.Printf("Waiting for re-election (30 seconds)...")
	time.Sleep(30 * time.Second)

	log.Printf("Starting node %s...", nodeName)
	cmd = exec.Command("docker", "start", containerID)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}
	log.Printf("Node started successfully")

	log.Printf("Waiting for node to be ready...")
	if err := waitForNodeReady(clientset, nodeName); err != nil {
		log.Fatalf("Failed waiting for node: %v", err)
	}
	log.Printf("Node is ready")
}
