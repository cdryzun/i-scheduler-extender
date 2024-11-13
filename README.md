# k8s 自定义调度器逻辑之 Scheduler Extender

通过 Scheduler Extender 扩展原有调度器，以实现自定义调度逻辑。

手把手实现一个简单的扩展调度器。

功能如下：

* 1）过滤阶段：仅调度到带有 Label `priority.lixueduan.com` 的节点上
* 2）打分阶段：直接将节点上  Label `priority.lixueduan.com` 的值作为得分
  * 比如某节点携带 Label `priority.lixueduan.com=50` 则打分阶段该节点则是 50 分

Scheduler Extender 系列完整内容见：[K8s 自定义调度器 Part1：通过 Scheduler Extender 实现自定义调度逻辑](https://www.lixueduan.com)


## 微信公众号：探索云原生

一个云原生打工人的探索之路，专注云原生，Go，坚持分享最佳实践、经验干货。

扫描下面二维码，关注我即时获取更新~

![](https://img.lixueduan.com/about/wechat/qrcode_search.png)


## Scheduler Extender 规范

Scheduler Extender 通过 HTTP 请求的方式，将调度框架阶段中的调度决策委托给外部的调度器，然后将调度结果返回给调度框架。

我们只需要实现一个 HTTP 服务，然后通过配置文件将其注册到调度器中，就可以实现自定义调度器。

通过 Scheduler Extender 扩展原有调度器一般分为以下两步：

* 1）创建一个 HTTP 服务，实现对应接口
* 2）修改调度器配置 KubeSchedulerConfiguration，增加 extenders 相关配置



外置调度器可以影响到三个阶段：

* Filter：调度框架将调用 Filter 函数，过滤掉不适合被调度的节点。

* Priority：调度框架将调用 Priority 函数，为每个节点计算一个优先级，优先级越高，节点越适合被调度。

* Bind：调度框架将调用 Bind 函数，将 Pod 绑定到一个节点上。

Filter、Priority、Bind 三个阶段分别对应三个 HTTP 接口，三个接口都接收 POST 请求，各自的请求、响应结构定义在这里：[#kubernetes/kube-scheduler/extender/v1/types.go](https://github.com/kubernetes/kube-scheduler/blob/master/extender/v1/types.go)

在这个 HTTP 服务中，我们可以实现上述阶段中的**任意一个或多个阶段的接口**，来定制我们的调度需求。


## 编译部署
```bash
make build-image
```
部署到集群

```bash 
kubectl apply -f deploy/manifest.yaml
```

确认服务正常运行

```bash
[root@scheduler-1 ~]# kubectl -n kube-system get po|grep i-scheduler-extender
i-scheduler-extender-f9cff954c-dkwz2   2/2     Running   0          1m
```



## 测试

### 启动测试 Pod

创建一个 Deployment 并指定使用上一步中部署的 Scheduler，然后测试会调度到哪个节点上。
```bash
kubectl apply -f deploy/deploy-test.yaml
```
创建之后 Pod 会一直处于 Pending 状态

```bash
[root@scheduler-1 lixd]# k get po
NAME                    READY   STATUS    RESTARTS   AGE
test-58794bff9f-ljxbs   0/1     Pending   0          17s

```

查看具体情况

```bash
[root@scheduler-1]# k describe po test-58794bff9f-ljxbs
Events:
  Type     Reason            Age                From                  Message
  ----     ------            ----               ----                  -------
  Warning  FailedScheduling  99s                i-scheduler-extender  all node do not have label priority.lixueduan.com
  Warning  FailedScheduling  95s (x2 over 97s)  i-scheduler-extender  all node do not have label priority.lixueduan.com
```

可以看到，是因为 Node 上没有我们定义的 Label，因此都不满足条件，最终 Pod 就一直 Pending 了。



### 给节点添加 Label

由于我们实现的 Filter 逻辑是需要 Node 上有`priority.lixueduan.com` 才会用来调度，否则直接会忽略。



理论上，只要给任意一个 Node 打上 Label 就可以了。

```bash
[root@scheduler-1 install]# k get node
NAME          STATUS   ROLES           AGE     VERSION
scheduler-1   Ready    control-plane   4h34m   v1.27.4
scheduler-2   Ready    <none>          4h33m   v1.27.4
[root@scheduler-1 install]# k label node scheduler-1 priority.lixueduan.com=10
node/scheduler-1 labeled
```

再次查看 Pod 状态

```bash
[root@scheduler-1 lixd]# k get po -owide
NAME                    READY   STATUS    RESTARTS   AGE    IP               NODE          NOMINATED NODE   READINESS GATES
test-58794bff9f-ljxbs   1/1     Running   0          104s   172.25.123.201   scheduler-1   <none>           <none>
```

已经被调度到 node1 上了，查看详细日志

```bash
[root@scheduler-1 install]# k describe po test-7f7bb8f449-w6wvv
Events:
  Type     Reason            Age                  From                  Message
  ----     ------            ----                 ----                  -------
  Warning  FailedScheduling  116s                 i-scheduler-extender  0/2 nodes are available: preemption: 0/2 nodes are available: 2 No preemption victims found for incoming pod.
  Warning  FailedScheduling  112s (x2 over 115s)  i-scheduler-extender  0/2 nodes are available: preemption: 0/2 nodes are available: 2 No preemption victims found for incoming pod.
  Normal   Scheduled         26s                  i-scheduler-extender  Successfully assigned default/test-58794bff9f-ljxbs to scheduler-1
```

可以看到，确实是 i-scheduler-extender 这个调度器在处理，调度到了 node1.



### 多节点排序

我们实现的 Score 是根据 Node 上的 `priority.lixueduan.com` 对应的 Value 作为得分的，因此调度器会**优先考虑**调度到 Value 比较大的一个节点。

> 因为 Score 阶段也有很多调度插件，Scheduler 会汇总所有得分，最终才会选出结果，因此这里的分数也是仅供参考，不能完全控制调度结果。



给 node2 也打上 label，value 设置为 20

```bash
[root@scheduler-1 install]# k get node
NAME          STATUS   ROLES           AGE     VERSION
scheduler-1   Ready    control-plane   4h34m   v1.27.4
scheduler-2   Ready    <none>          4h33m   v1.27.4
[root@scheduler-1 install]# k label node scheduler-2 priority.lixueduan.com=20
node/scheduler-2 labeled
```

然后更新 Deployment ，触发创建新 Pod ，测试调度逻辑。

```bash
[root@scheduler-1 lixd]# kubectl rollout restart deploy test
deployment.apps/test restarted
```

因为 Node2 上的 priority 为 20，node1 上为 10，那么理论上会调度到 node2 上。

```bash
[root@scheduler-1 lixd]# k get po -owide
NAME                    READY   STATUS    RESTARTS   AGE   IP             NODE          NOMINATED NODE   READINESS GATES
test-84fdbbd8c7-47mtr   1/1     Running   0          38s   172.25.0.162   scheduler-1   <none>           <none>
```

***结果还是调度到了 node1，为什么呢？***

这就是前面提到的：因为 Extender 仅作为一个额外的调度插件接入，**Prioritize 接口返回得分最终 Scheduler 会将其和其他插件打分合并之后再选出最终节点**，因此 Extender 想要完全控制调度结果，只能在 Filter 接口中实现，过滤掉不满足条件的节点，并对剩余节点进行打分，最终 Filter 接口只返回得分最高的那个节点，从而实现完全控制调度结果。

> ps：即之前的 Filter OnlyOne 实现，可以在 KubeSchedulerConfiguration 中配置不同的 path 来调用不同接口进行测试。



修改 KubeSchedulerConfiguration 配置，

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: i-scheduler-extender
  namespace: kube-system
data:
  i-scheduler-extender.yaml: |
    apiVersion: kubescheduler.config.k8s.io/v1
    kind: KubeSchedulerConfiguration
    profiles:
      - schedulerName: i-scheduler-extender
    leaderElection:
      leaderElect: false
    extenders:
    - urlPrefix: "http://localhost:8080"
      enableHTTPS: false
      filterVerb: "filter_onlyone"
      prioritizeVerb: "prioritize"
      bindVerb: "bind"
      weight: 1
      nodeCacheCapable: true
```

修改点：

```yaml
filterVerb: "filter_onlyone"
```

Path 从 filter 修改成了 filter\_onlyone，这里的 path 和前面注册服务时的路径对应：

```go
    http.HandleFunc("/filter", h.Filter)
    http.HandleFunc("/filter_onlyone", h.FilterOnlyOne) // Filter 接口的一个额外实现
```

修改后重启一下 Scheduler

```bash
kubectl -n kube-system rollout restart deploy i-scheduler-extender
```

再次更新 Deployment 触发调度

```bash
[root@scheduler-1 install]# k rollout restart deploy test
deployment.apps/test restarted
```

这样应该是调度到 node2 了，确认一下

```bash
[root@scheduler-1 lixd]# k get po -owide
NAME                    READY   STATUS    RESTARTS   AGE   IP             NODE          NOMINATED NODE   READINESS GATES
test-849f549d5b-pbrml   1/1     Running       0          12s   172.25.0.166     scheduler-2   <none>           <none>
```

现在我们更新 Node1 的 label，改成 30

```bash
k label node scheduler-1 priority.lixueduan.com=30 --overwrite
```

再次更新 Deployment 触发调度

```bash
[root@scheduler-1 install]# k rollout restart deploy test
deployment.apps/test restarted
```

这样应该是调度到 node1 了，确认一下

```bash
[root@scheduler-1 lixd]# k get po -owide
NAME                    READY   STATUS        RESTARTS   AGE   IP               NODE          NOMINATED NODE   READINESS GATES
test-69d9ccb877-9fb6t   1/1     Running       0          5s    172.25.123.203   scheduler-1   <none>           <none>
```

说明修改 Filter 方法实现之后，确实可以直接控制调度结果。

