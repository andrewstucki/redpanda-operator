package main

import "github.com/redpanda-data/redpanda-operator/acceptance/framework/tablegenerator"

var providers = []string{"eks", "aks", "gke", "k3d"}

func main() {
	tablegenerator.RunGenerator("README.md", "<!-- insert snippet -->", "###", providers...)
}
