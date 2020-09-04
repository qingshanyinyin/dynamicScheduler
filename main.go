package main

import (
	"dynamicScheduler/prom"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"strconv"
	"os"
	"path/filepath"
	//"sync"
	"time"
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"encoding/json"
	//"k8s.io/apimachinery/pkg/types"
)



var(

	FitSelectorAndAlreadyLabelPresureNodes []*v1.Node           //符合标签选择器，
	FitSelectorAndAlreadyLabelPresureNodeNames []string   //集群内已经上压力的node Names
	FitSelectorNodes []*v1.Node                     //根据标签选择器获取的当前node slice

)

type Mappintstruct struct {
	Node *v1.Node
	tag string
}


func main() {

	stopCh := make(chan struct{})
	/*
		k8sClient初始化
	*/
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, "src", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	/*
		prometheus 客户都初始化
	*/
	client, err := api.NewClient(api.Config{
		Address: "http://121.40.224.66:49090",
	})
	if err != nil {
		klog.Error("Error creating client: %v\n", err)
		os.Exit(1)
	}
	v1api := promv1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	/*
		筛选资源使用率负载较高的节点(cpu,mem,node_load)，打上对应的标签(status=presure)
	*/
	//1.NewSharedInfomerFactory
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	//2.client 访问prometheus接口查询 节点使用指标
	resultFromPromSilceMap, _ := prom.QueryRebuild(v1api, ctx, prom.Node_load1, time.Now())
	//3. 统计当前从promtheus上获取到的metrics超出阈值的NodeNames
	UpperPresureNodeNames:=CountUpperPresureNodeFromProm(resultFromPromSilceMap, 1.00)

	//4. 统计当前k8s 集群已经打上usage超出阈值的节点Nodenames ;修改 AlreadyLabelPromStatusNodeNames=[]string{} ;selectorNodes []*v1.Node ;SelectorAndPreSureNodes []*v1.Node
	HandlerAlreadyLabelPromStatusNodesInformer(stopCh, factory)
	time.Sleep(time.Second*1)
	//5. 给超出阈值的节点节点打上label
	//ReadyForLabelPresure:=[]*v1.Node{}
	//ReadyForLabelNil:=[]*v1.Node{}


	//FitSelectorNodeNamesSet:=utils.New()
	//UppperPresureNodeNamesSet:=utils.New()
	//
	//FitSelectorNodeNames:=[]string{}
	//for i:=0;i<len(FitSelectorNodes);i++{
	//	FitSelectorNodeNames=append(FitSelectorNodeNames,FitSelectorNodes[i].Name)
	//}
	//UppperPresureNodeNamesSet.Add(UpperPresureNodeNames)
	//FitSelectorNodeNamesSet.Add(FitSelectorNodeNames)
	//
	//fmt.Println(UppperPresureNodeNamesSet.List(),"up")
	//fmt.Println(FitSelectorNodeNamesSet.List(),"all")
	//ReadyForLabelNilNodeNames:=UppperPresureNodeNamesSet.Minus(FitSelectorNodeNamesSet)
	//fmt.Println(ReadyForLabelNilNodeNames.List())




	ReadyForLabelPresure:=[]*v1.Node{}
	ReadyForLabelNil:=[]*v1.Node{}


	for _,v:=range(FitSelectorNodes){
		key:=v.Name
		if IsExitArray(key,UpperPresureNodeNames){
			ReadyForLabelPresure=append(ReadyForLabelPresure,v)
		}else{
			ReadyForLabelNil=append(ReadyForLabelNil,v)
		}
	}
	fmt.Println(len(ReadyForLabelPresure))
	fmt.Println("ReadynIl",len(ReadyForLabelNil))

	<-stopCh
}


func IsExitArray(value string,arry []string ) bool{
	for _,v:=range(arry){
		if v ==value{
			return true
		}
	}
	return false

}



// 根据n阈值，筛选超出阈值node节点（master和slave)
func CountUpperPresureNodeFromProm(resultFromPromSilceMap []map[string]string, threshold float64) []string{
	presureNodesNameFromProm:=[]string{}
	for i:=0;i<len(resultFromPromSilceMap);i++{
		currentMetrics,_:=strconv.ParseFloat(resultFromPromSilceMap[i]["value"],64)
		fmt.Println(resultFromPromSilceMap[i]["instance"],currentMetrics)
		// 判断负载值是否
		if currentMetrics>threshold{
			presureNodesNameFromProm =append(presureNodesNameFromProm,resultFromPromSilceMap[i]["instance"])
		}
	}
	return presureNodesNameFromProm
}



//统计处理已经打了压力标签的node 节点silce，和当前符合selector节点的[]*v1.Node
//修改全局变量 Nodes []*v1.Node,	AlreadyLabelPromStatusNodeNames=[]string{}
func HandlerAlreadyLabelPromStatusNodesInformer(stopCh <-chan struct{},factory informers.SharedInformerFactory ){
	nodeInformer := factory.Core().V1().Nodes()
	// 2.1 开启node informer
	go nodeInformer.Informer().Run(stopCh)
	//从k8s中同步list node
	if !cache.WaitForCacheSync(stopCh, nodeInformer.Informer().HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}
	//node selector
	nodeOthersSlice := []string{"others"}
	nodeOthersSlice2 := []string{"presure"}

	//返回打了others和persure节点*[]v1.Node
	selector, _ := labels.NewRequirement("type", selection.In, nodeOthersSlice)
	selector2, _ := labels.NewRequirement("status", selection.In, nodeOthersSlice2)

	go func() {
		for {
			FitSelectorNodes,_ = nodeInformer.Lister().List(labels.NewSelector())
			FitSelectorAndAlreadyLabelPresureNodes, _ = nodeInformer.Lister().List(labels.NewSelector().Add(*selector,*selector2))
			FitSelectorAndAlreadyLabelPresureNodeNames =[]string{}
			for i := 0; i < len(FitSelectorAndAlreadyLabelPresureNodes); i++ {
				FitSelectorAndAlreadyLabelPresureNodeNames=append(FitSelectorAndAlreadyLabelPresureNodeNames, FitSelectorAndAlreadyLabelPresureNodes[i].Name)
			}
			time.Sleep(time.Second * 5)
		}
	}()

}



//label node by patch
func PatchNode(clientset *kubernetes.Clientset,ctx context.Context,selection string,Nodes []*v1.Node){
    //const selection pressure or nil
	var labelvaule  interface{}
	labelkey :="status"
	switch selection{
		//去掉标签
		case "nil":
			labelvaule = nil
		//打上标签
		case "persure":
			labelvaule =fmt.Sprintf("%s",selection)
	}
	patchTemplate:=map[string]interface{}{
		"metadata":map[string]interface{}{
			"labels":map[string]interface{}{
				labelkey:labelvaule,
			},
		},
	}
	patchdata,_:=json.Marshal(patchTemplate)
	for i:=0;i<len(Nodes);i++{
		//master节点不做label处理
		if _,ok:=Nodes[i].Labels["node-role.kubernetes.io/master"];!ok{
			_,err:=clientset.CoreV1().Nodes().Patch(ctx,Nodes[i].Name,types.StrategicMergePatchType,patchdata,metav1.PatchOptions{})
			if err!=nil{
				klog.Error("给节点%s打标签失败，错误为%s",Nodes[i],err)
			}
		}
	}
}


func homeDir() string {
	if h := os.Getenv("GOPATH"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

