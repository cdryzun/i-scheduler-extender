package extender

import (
	v1 "k8s.io/api/core/v1"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"sort"
)

// Filter 过滤掉不满足条件的节点
func (ex *Extender) Filter(args extenderv1.ExtenderArgs) *extenderv1.ExtenderFilterResult {
	nodes := make([]v1.Node, 0)
	nodeNames := make([]string, 0)

	for _, node := range args.Nodes.Items {
		_, ok := node.Labels[Label]
		if !ok { // 排除掉不带指定标签的节点
			continue
		}
		nodes = append(nodes, node)
		nodeNames = append(nodeNames, node.Name)
	}

	args.Nodes.Items = nodes

	return &extenderv1.ExtenderFilterResult{
		Nodes:     args.Nodes, // 当 NodeCacheCapable 设置为 false 时会使用这个值
		NodeNames: &nodeNames, // 当 NodeCacheCapable 设置为 true 时会使用这个值
	}
}

// FilterOnlyOne 过滤掉不满足条件的节点,并将其余节点打分排序，最终只返回得分最高的节点以实现完全控制调度结果
func (ex *Extender) FilterOnlyOne(args extenderv1.ExtenderArgs) *extenderv1.ExtenderFilterResult {
	// 过滤掉不满足条件的节点
	nodeScores := &NodeScoreList{NodeList: make([]*NodeScore, 0)}

	for _, node := range args.Nodes.Items {
		_, ok := node.Labels[Label]
		if !ok { // 排除掉不带指定标签的节点
			continue
		}
		// 对剩余节点打分
		score := ComputeScore(node)
		nodeScores.NodeList = append(nodeScores.NodeList, &NodeScore{Node: node, Score: score})
	}

	// 排序
	sort.Sort(nodeScores)
	// 然后取最后一个，即得分最高的节点，这样由于 Filter 只返回了一个节点，因此最终肯定会调度到该节点上
	m := (*nodeScores).NodeList[len((*nodeScores).NodeList)-1]

	// 组装一下返回结果
	args.Nodes.Items = []v1.Node{m.Node}

	return &extenderv1.ExtenderFilterResult{
		Nodes:     args.Nodes,
		NodeNames: &[]string{m.Node.Name},
	}
}
