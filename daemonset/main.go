package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	v1alpha1 "github.com/cloud-native-skunkworks/ubuntu-operator/api/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Module struct {
	Name  string `json:"name"`
	Flags string `json:"flags"`
}

type RelayMessage struct {
	Type           string   `json:"type"` // "Request | Response"
	HostName       string   `json:"hostname"`
	DesiredModules []Module `json:"desiredModules"`
	ActualModules  []Module `json:"actualModules"`
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	myFlags         arrayFlags
	minWatchTimeout = 5 * time.Minute
	masterURL       = flag.String("masterURL", "", "masterURL")
	kubeconfig      = flag.String("kubeconfig", "", "kubeconfig")
	socketPath      = flag.String("socketPath", "", "socketPath")
)

func buildClient(config *rest.Config) *rest.RESTClient {
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &v1alpha1.GroupVersion
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		panic(err)
	}

	return client
}

func fetchUbuntuKernelModuleCR(restClient *rest.RESTClient) (v1alpha1.UbuntuKernelModuleList, error) {
	result := v1alpha1.UbuntuKernelModuleList{}
	err := restClient.Get().Resource("ubuntukernelmodules").Do(context.TODO()).Into(&result)

	return result, err
}

func main() {

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)

	flag.Var(&myFlags, "module", "Module and args e.g. --module=nvme_core --module=rfcomm=foo")
	flag.Parse()
	// Setup KubeClient -----------------------------------------------------------------------------

	kubeCfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	v1alpha1.AddToScheme(scheme.Scheme)
	restClient := buildClient(kubeCfg)

	// Setup Kmod ------------------------------------------------------------------------------------

	var desiredModules []Module
	for _, module := range myFlags {
		fmt.Println("Desired module:", module)
		s := strings.Split(module, "=")
		m := Module{Name: s[0]}
		if len(s) > 1 {
			m.Flags = s[1]
		}
		desiredModules = append(desiredModules, m)
	}

	// This allows the daemonset to pass through module lists
	if os.Getenv("MODULE_LIST") != "" {
		envList := strings.Split(os.Getenv("MODULE_LIST"), ",")
		for _, module := range envList {
			myFlags = append(myFlags, module)
		}
	}

	// ------------------------------------------------------------------------------------------------
	if len(myFlags) == 0 {
		fmt.Printf("No modules specified. Exiting.\n")
		return
	}
	if *socketPath == "" {
		fmt.Printf("No --socketPath set")
		return
	}

	for {
		fmt.Printf("Using socketpath %s", socketPath)
		c, err := net.Dial("unix", *socketPath)
		if err != nil {
			panic(err.Error())
		}
		defer c.Close()
		// // Check that the CR exists before we start polling
		// li, err := fetchUbuntuKernelModuleCR(restClient)
		// if err != nil || len(li.Items) == 0 {
		// 	log.Warningf("No UbuntuKernelModule CR found. Waiting for it to be created.")
		// 	continue
		// }
		// TODO:
		//
		// Currently the architecture is for a single UbuntuKernelModule CR.
		// This may need to change in the future
		//
		sendMessage := RelayMessage{
			Type:           "Request",
			DesiredModules: desiredModules,
		}

		b, err := json.Marshal(sendMessage)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		b = append(b, '\n')

		_, err = c.Write(b)
		if err != nil {
			println(err.Error())
			continue
		}

		reader := bufio.NewReader(c)
		data, err := reader.ReadBytes('\n')
		if err != nil {
			println(err.Error())
			continue
		}

		data = data[:len(data)-1]

		var msg RelayMessage
		err = json.Unmarshal(data, &msg)
		if err != nil {
			fmt.Println("Error unmarshalling message:", err)
			return
		}

		switch msg.Type {
		case "Response":
			fmt.Printf("Response: %s", msg.ActualModules)

			// Write back the changes
			li, err := fetchUbuntuKernelModuleCR(restClient)
			if err != nil || len(li.Items) == 0 {
				log.Warningf("No UbuntuKernelModule CR found. Waiting for it to be created.")
				continue
			}
			//TODO:
			// Only interacting with the first CR

			kernelModuleCR := li.Items[0]

			var modules []v1alpha1.Module
			for _, mods := range msg.ActualModules {

				modules = append(modules, v1alpha1.Module{
					Name:  mods.Name,
					Flags: mods.Flags,
				})

			}

			kernelModuleCR.Status.Nodes = append(kernelModuleCR.Status.Nodes, v1alpha1.Node{
				Name:    msg.HostName,
				Modules: modules,
			})

			restClient.Put().Resource("ubuntukernelmodules").Body(v1alpha1.UbuntuKernelModuleList{
				Items: []v1alpha1.UbuntuKernelModule{kernelModuleCR},
			}).Do(context.TODO())

		}

		time.Sleep(time.Second * 30)
	}
}
