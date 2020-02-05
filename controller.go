package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	//"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/kubernetes/scheme"
	apicorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	networklisters "k8s.io/client-go/listers/networking/v1"
	//v1 "k8s.io/client-go/pkg/apis/networking/v1"
	v1Networking "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	secretSyncAnnotation       = "eightypercent.net/secretsync"
	netowrkPoliySyncAnnotation = "eightypercent.net/networkPolicysync"

	secretSyncSourceNamespace = "secretsync"
	secretSyncKey             = "secretSyncKey"
	networkPolicySyncKey      = "networkPolicySyncKey"
	podSyncKey                = "podSyncKey"
)

var namespaceBlacklist = map[string]bool{
	"kube-public":             true,
	"kube-system":             true,
	secretSyncSourceNamespace: true,
}

type TGIKController struct {
	secretGetter       corev1.SecretsGetter
	secretLister       listercorev1.SecretLister
	secretListerSynced cache.InformerSynced

	podGetter       corev1.PodsGetter
	podLister       listercorev1.PodLister
	podListerSynced cache.InformerSynced

	namespaceGetter       corev1.NamespacesGetter
	namespaceLister       listercorev1.NamespaceLister
	namespaceListerSynced cache.InformerSynced

	networkPoliciesLister networklisters.NetworkPolicyLister
	networkPoliciesSynced cache.InformerSynced

	queue workqueue.RateLimitingInterface
}

func NewTGIKController(client *kubernetes.Clientset,
	secretInformer informercorev1.SecretInformer,
	namespaceInformer informercorev1.NamespaceInformer,
	kubeInformerFactory kubeinformers.SharedInformerFactory) *TGIKController {

	networkPolicyInformer := kubeInformerFactory.Networking().V1().NetworkPolicies()
	podInformer := kubeInformerFactory.Core().V1().Pods()

	c := &TGIKController{
		secretGetter:       client.CoreV1(),
		secretLister:       secretInformer.Lister(),
		secretListerSynced: secretInformer.Informer().HasSynced,

		podGetter:       client.CoreV1(),
		podLister:       podInformer.Lister(),
		podListerSynced: podInformer.Informer().HasSynced,

		namespaceGetter:       client.CoreV1(),
		namespaceLister:       namespaceInformer.Lister(),
		namespaceListerSynced: namespaceInformer.Informer().HasSynced,

		networkPoliciesLister: networkPolicyInformer.Lister(),
		networkPoliciesSynced: networkPolicyInformer.Informer().HasSynced,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "secretsync"),
	}

	networkPolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Print("NetWorkPolicy  added")
			c.ScheduleNetworkPolicySync()
		},
	})

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Print("Pod  added")
			c.SchedulePodSync()
		},
	})
	c.podListerSynced = podInformer.Informer().HasSynced

	return c

}

func cmdExec(kubectlCMD string) {
	kubectlCMD = "foo"
	log.Print("cmdEec: started: ")
	cmd := exec.Command("kubectl", "exec", "  ")
	cmd.Stdin = strings.NewReader(" label-demo -- wget -q0 - --timeout=2 http://bar.default ")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("in all caps: %q\n", out.String())

	log.Print("cmdEec: finished: ")
}
func (c *TGIKController) testIngress(rowIngressPolicies []*v1Networking.NetworkPolicy) {

	//var networkPolicies []*v1.NetworkPolicy
	for _, oneNetworkPolicy := range rowIngressPolicies {
		//	log.Print("getNetworkPolicyInNS2:", oneNetworkPolicy)
		//	oneNetworkPolicy.
		if _, ok := oneNetworkPolicy.Annotations[netowrkPoliySyncAnnotation]; ok {
			//networkPolicies = append(networkPolicies, oneNetworkPolicy)
			log.Print("getNetworkPolicyInNS: done ", oneNetworkPolicy.GetName())
			ingressRules := oneNetworkPolicy.Spec.Ingress
			policyType := oneNetworkPolicy.Spec.PolicyTypes
			applyToPods := oneNetworkPolicy.Spec.PodSelector
			log.Print("yahoooooooooo0", applyToPods.MatchLabels)

			log.Print("applyToPods.MatchLabels :", applyToPods.MatchLabels)

			podsUnderTest := c.getPodNamesUnderTestFromNPC(&applyToPods)

			for _, onePodUnderTestName := range podsUnderTest {
				cmdExec(onePodUnderTestName)
			}
			for _, policy := range policyType {
				log.Print("***Policy Type is :", policy)
			}
			for _, rule := range ingressRules {
				listOfSrcs := rule.From
				for _, fromOneSrc := range listOfSrcs {
					//					log.Printf("ingress rules -> PodSelector: %v", fromOneSrc.PodSelector.MatchLabels)
					if fromOneSrc.NamespaceSelector != nil {
						log.Printf("ingress rules -> NamespaceSelector: %v", fromOneSrc.NamespaceSelector.MatchLabels)
					}
					if fromOneSrc.IPBlock != nil {
						log.Printf("ingress rules -> CIDR: %v", fromOneSrc.IPBlock.CIDR)
					}
					if fromOneSrc.IPBlock != nil {
						log.Printf("ingress rules -> Except: %v", fromOneSrc.IPBlock.Except)
					}
				}

				allowedPorts := rule.Ports
				for _, allowedPortsValues := range allowedPorts {
					if allowedPortsValues.Protocol != nil {

						protoc := allowedPortsValues.Protocol
						log.Print("ingress rules -> port protocol: ", protoc)
					}
					if allowedPortsValues.Port != nil {
						log.Printf("ingress rules -> Port number: %v", allowedPortsValues.Port.String())
					}
				}
			}

			//egressRules := oneNetworkPolicy.Spec.Egress

			// for _, egressRule := range egressRules {
			// 	toDests := egressRule.To
			// 	for _, toOneDist := range toDests {

			// 		log.Print("egress rules -> PodSelector: ", toOneDist.PodSelector.MatchLabels)
			// 		log.Print("egress rules -> NamespaceSelector: ", toOneDist.NamespaceSelector.MatchLabels)
			// 		log.Print("egress rules -> CIDR: ", toOneDist.IPBlock.CIDR)
			// 		log.Print("egress rules -> except: ", toOneDist.IPBlock.Except)
			// 	}

			// 	allowedIcomingPorts := egressRule.Ports
			// 	for _, allowedIcomingPortsValues := range allowedIcomingPorts {
			// 		log.Print("egress rules -> port protocol: ", allowedIcomingPortsValues.Protocol)
			// 		log.Print("egress rules -> Port number: ", allowedIcomingPortsValues.Port.String())
			// 	}
			// }
			log.Print("getNetworkPolicyInNS: done ", oneNetworkPolicy.Spec.Ingress)

		}
	}
}

func (c *TGIKController) getNetworkPolicyInNS(ns string) ([]*v1Networking.NetworkPolicy, error) {
	log.Print("getNetworkPolicyInNS: start grapping network policies inside the name space ")
	rawNCPs, err := c.networkPoliciesLister.NetworkPolicies(ns).List(labels.Everything())
	//log.Print("getNetworkPolicyInNS1:", rawNCPs)
	if err != nil {
		return nil, err
	}

	return rawNCPs, nil
}

func (c *TGIKController) doSync() error {
	log.Printf("Starting doSync for Network Policy ")

	rawNCPs, err := c.getNetworkPolicyInNS(apicorev1.NamespaceDefault)
	if err != nil {
		return err
	}
	log.Print("Finishing doSync")
	log.Print("Testing Ingress Polices using port scanner")
	// we will login into every pod[] let's called it testerPods who suppose to ping the pod under test
	// then try to connect to from the testerPods to the podUnderTest and store the result of connection for every port
	// then using the port, protocol
	// ex : podsUnderTest[] : selector by name space [default] and labels role: db as podsUnderTest[]
	//    : testersPods[] : selected by :  podselector &  namespace labels and pod labels
	// 	- ipBlock:
	// 	cidr: 172.17.0.0/16
	// 	except:
	// 	- 172.17.1.0/24
	// - namespaceSelector:
	// 	matchLabels:
	// 	  project: myproject
	// - podSelector:
	// 	matchLabels:
	// 	  role: frontend
	//    : for each testersPods do :
	//    						login with paramerters of of these pods and  send ping/wget/curl to every -> podsUnderTest[]
	//							send results to collectorSideCar

	// 	  :
	// 1- Test Ingress Policies
	c.testIngress(rawNCPs)
	// 2- Test Egress Polices
	log.Print("Done ...")

	return err
}
func (c *TGIKController) buildConnection(testersPods map[string]string, podsUnderTest map[string]string) {

	// for each
	// get map[string]string
	// for _, testersPod := range testersPods {
	// 	// login into pod
	// 	// 1-Verify that the Container is running:
	// 	// kubectl get pod shell-demo
	// 	// 2 Get a shell to the running Container:
	// 	//kubectl exec -it shell-demo -- /bin/bash
	// 	// kubectl exec -it init-demo -- /bin/bash
	// 	// wget -qO- 10.244.0.5:9376
	// 	// kubectl attach
	// 	// kubectl exec - it foo -- wget -q0 - --timeout=2 http://bar.default -> wget:downloadout

	// }
	return

}

func (c *TGIKController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *TGIKController) processNextWorkItem() bool {
	// pull the next work item from queue.  It should be a key we use to lookup
	// something in a cache
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// you always have to indicate to the queue that you've completed a piece of
	// work
	defer c.queue.Done(key)

	// do your work on the key.  This method will contains your "do stuff" logic
	err := c.doSync()
	if err == nil {
		// if you had no error, tell the queue to stop tracking history for your
		// key. This will reset things like failure counts for per-item rate
		// limiting
		c.queue.Forget(key)
		return true
	}

	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring
	runtime.HandleError(fmt.Errorf("doSync failed with: %v", err))

	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	c.queue.AddRateLimited(key)

	return true
}

func (c *TGIKController) ScheduleSecretSync() {
	c.queue.Add(secretSyncKey)
}

func (c *TGIKController) ScheduleNetworkPolicySync() {
	c.queue.Add(networkPolicySyncKey)
}
func (c *TGIKController) SchedulePodSync() {
	c.queue.Add(podSyncKey)
}
func (c *TGIKController) Run(stop <-chan struct{}) {
	var wg sync.WaitGroup

	defer func() {
		// make sure the work queue is shut down which will trigger workers to end
		log.Print("shutting down queue")
		c.queue.ShutDown()

		// wait on the workers
		log.Print("shutting down workers")
		wg.Wait()

		log.Print("workers are all done")
	}()

	log.Print("waiting for cache sync")
	if !cache.WaitForCacheSync(
		stop,
		c.secretListerSynced,
		c.podListerSynced,
		c.namespaceListerSynced) {
		log.Print("timed out waiting for cache sync")
		return
	}
	log.Print("caches are synced")

	go func() {
		// runWorker will loop until "something bad" happens. wait.Until will
		// then rekick the worker after one second.
		wait.Until(c.runWorker, time.Second, stop)
		// tell the WaitGroup this worker is done
		wg.Done()
	}()

	// wait until we're told to stop
	log.Print("waiting for stop signal")
	<-stop
	log.Print("received stop signal")
}

// filterPods returns pods based on their phase.
func filterPods(pods []*v1.Pod, phase v1.PodPhase) (ret []string) {
	//filteredPods := make([]*v1.Pod, len(pods))

	for i, pod := range pods {
		if phase == pods[i].Status.Phase {
			ret = append(ret, pod.Name)
		}
	}
	return ret
}

//filterPods(pods, v1.PodSucceeded)

// getStatus returns no of succeeded and failed pods running a job
// func getStatus(pods []*v1.Pod) (succeeded, failed int32) {
// 	succeeded = (filterPods(pods, v1.PodSucceeded))
// 	failed = int32(filterPods(pods, v1.PodFailed))
// 	return
// }

// getStatus returns no of succeeded and failed pods running a job

func (c TGIKController) getPodNamesUnderTestFromNPC(podSelector *metav1.LabelSelector) []string {

	selector, err := metav1.LabelSelectorAsSelector(podSelector)
	log.Print("selectorselectorselector", selector)

	if err != nil {
		fmt.Errorf("couldn't find pods under test selector: %v", err)
	}

	podsUnderTestRaw, err := c.podLister.Pods(apicorev1.NamespaceDefault).List(selector)

	if err != nil {
		fmt.Errorf("error in getting pod", err)
	}
	podsUnderTest := filterPods(podsUnderTestRaw, v1.PodRunning)

	//if podsUnderTest[0] != nil {

	for _, pod := range podsUnderTest {

		log.Print("Found Pods:", pod)

	}

	return podsUnderTest
	//}
}
