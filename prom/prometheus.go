package prom

import (
	"context"
	"fmt"
	"k8s.io/klog"
	"os"
	"strings"
	"time"
	//"unicode"

	//"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)


// promethues merics
var (
	promethesuJob="测试环境k8s资源节点监控"
	Node_load15 string = fmt.Sprintf("node_load15{job='%s'}", promethesuJob)
	Node_load5 string = fmt.Sprintf("node_load5{job='%s'}", promethesuJob)
	Node_load1 string = fmt.Sprintf("node_load1{job='%s'}", promethesuJob)
)



//prometheus query 重构prometheus query
func QueryRebuild(v1api v1.API,ctx context.Context, query string, ts time.Time) ([]map[string]string ,bool){
	result, warnings, err := v1api.Query(ctx, query, ts)
	if err != nil {
		klog.Error("Error querying Prometheus: %s\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		klog.Error("Warnings: %v\n", warnings)
	}

	if result.String() !=""{
		resultslice := strings.Split(result.String(), "\n")
		resultSliceMap := ConvertResultDataType(resultslice)
		return resultSliceMap ,true
	}else{
		klog.Errorf("查询promql有问题,promsql返回结果为:nil,%v",result)
		return  nil ,false
	}
}


// clinet_golang 访问prometheus返回结果为string
// 通过strings的切换，转换为[]map[string]string 类型
func ConvertResultDataType(resultslice []string) []map[string]string {
	var resultSilceMap []map[string]string

	for i := 0; i < len(resultslice); i++ {

		s := string(resultslice[i])
		//up{instance="cn-hangzhou.172.16.94.142", job="k8s资源节点监控"} => 1 @[1599043436.489]
		// 1.匹配{}中的内容换成数组
		start := strings.Index(s, "{")
		end := strings.Index(s, "}")
		value1 := strings.TrimSpace(s[start+1 : end])
		value1 = strings.Map(mapping, value1)
		metadataslice := strings.Fields(value1)

		// 2.匹配=> 之后的值为value
		start2 := strings.Index(s, "=>")
		end2 := strings.Index(s, "@")
		value2 := strings.TrimSpace(s[start2:end2])
		value2 = strings.Map(mapping, value2)
		//3.
		metadatamap := make(map[string]string)

		for i := 0; i < len(metadataslice); i++ {
			if i%2 == 0 {
				j := i
				key := strings.TrimSpace(metadataslice[j])
				value := strings.TrimSpace(metadataslice[j+1])
				metadatamap[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}

		metadatamap["value"] = strings.TrimSpace(value2)
		resultSilceMap = append(resultSilceMap, metadatamap)
	}
	return resultSilceMap
}

func mapping(r rune) rune {
	if r == '=' || r == ',' || r == '>'||r =='"' {
		return ' '
	}
	return r
}