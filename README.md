[toc]

# dynamic-Scheduler



### 背景

kubernetes调度策略默认根据node节点的request值（cpu和mem）进行调度，因此会存在如下痛点

- request设置太小，导致node节点上被分配太多的pod，面对流量高峰时，当前节点的资源使用率过高，导致一些pod 出现oom或者无法执行业务的情况。
- request接近于limits,导致node节点上分配的pod数量有限，不能够最大容量的分配pod，而本身设置limits值得原因大多是基于高峰资源，而实际业务大多数情况并未处于高峰状态
- 个性化原因：我们的业务模型创建的pod是自主性pod，因此一些好用的开源的动态调度策略组件并不满足需求，例如：[descheduler](https://github.com/kubernetes-sigs/descheduler)



### 



