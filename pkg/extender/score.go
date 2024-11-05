package extender

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"strconv"
)

type NodeScore struct {
	Node  v1.Node
	Score int64
}

type NodeScoreList struct {
	NodeList []*NodeScore
}

func (l NodeScoreList) Len() int {
	return len(l.NodeList)
}

func (l NodeScoreList) Swap(i, j int) {
	l.NodeList[i], l.NodeList[j] = l.NodeList[j], l.NodeList[i]
}

func (l NodeScoreList) Less(i, j int) bool {
	return l.NodeList[i].Score < l.NodeList[j].Score
}

func ComputeScore(node v1.Node) int64 {
	// 获取 Node 上的 Label 作为分数
	priorityStr, ok := node.Labels[Label]
	if !ok {
		klog.Errorf("node %q does not have label %s", node.Name, Label)
		return 0
	}

	priority, err := strconv.Atoi(priorityStr)
	if err != nil {
		klog.Errorf("node %q has priority %s are invalid", node.Name, priorityStr)
		return 0
	}
	return int64(priority)
}
