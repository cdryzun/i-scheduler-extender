package extender

import (
	"strconv"

	"k8s.io/klog/v2"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
)

// Prioritize 给 Pod 打分
// 注意：此处返回得分 Scheduler 会将其与其他插件打分合并后再选择节点，因此这里的逻辑不能完全控制最终的调度结果。
// 想要完全控制调度结果，只能在 Filter 接口中实现，过滤掉不满足条件的节点，并对剩余节点进行打分，最终 Filter 接口只返回得分最高的那个节点
func (ex *Extender) Prioritize(args extenderv1.ExtenderArgs) *extenderv1.HostPriorityList {
	// 初始化一个空的结果列表
	result := make(extenderv1.HostPriorityList, 0)

	// 检查 args.Nodes 是否为 nil
	if args.Nodes == nil {
		klog.Error("nodes is nil in extender args")
		return &result
	}

	// 检查 Items 是否为空
	if len(args.Nodes.Items) == 0 {
		klog.Error("no nodes available for prioritizing")
		return &result
	}

	for _, node := range args.Nodes.Items {
		// 获取 Node 上的 Label 作为分数
		priorityStr, ok := node.Labels[Label]
		if !ok {
			klog.Errorf("node %q does not have label %s", node.Name, Label)
			continue
		}

		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			klog.Errorf("node %q has priority %s are invalid", node.Name, priorityStr)
			continue
		}

		result = append(result, extenderv1.HostPriority{
			Host:  node.Name,
			Score: int64(priority),
		})
	}

	return &result
}
