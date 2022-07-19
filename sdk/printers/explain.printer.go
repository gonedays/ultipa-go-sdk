package printers

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"ultipa-go-sdk/sdk/structs"
)

type TreeNode struct {
	Explain    *structs.Explain
	ChildNodes []*TreeNode
}

var leveledList pterm.LeveledList

func PrintExplain(graphs []*structs.Explain) {
	if graphs == nil || len(graphs) == 0 {
		return
	}

	//tree := constructTree(graphs)

	explainChan := make(chan *structs.Explain, len(graphs))
	for _, explain := range graphs {
		explainChan <- explain
	}
	close(explainChan)
	tree := buildTree(explainChan)

	traverse(tree, 0)
	root := putils.TreeFromLeveledList(leveledList)

	pterm.DefaultTree.WithIndent(3).WithRoot(root).Render()
}

func constructTree(graphs []*structs.Explain) *TreeNode {
	if graphs == nil || len(graphs) == 0 {
		return &TreeNode{}
	}
	root := &TreeNode{
		Explain:    graphs[0],
		ChildNodes: []*TreeNode{},
	}
	var last *TreeNode
	for i, record := range graphs {
		if i == 0 {
			last = root
			continue
		}

		last.ChildNodes = append(last.ChildNodes, &TreeNode{
			Explain:    record,
			ChildNodes: []*TreeNode{},
		})
		if int(record.ChildrenNum) > 0 {
			lastIndex := len(last.ChildNodes) - 1
			last = last.ChildNodes[lastIndex]
		} else if i > 0 {
			last = root
		}
	}
	return root
}

func buildTree(graphs chan *structs.Explain) *TreeNode {
	if graphs == nil || len(graphs) == 0 {
		return nil
	}

	record := <-graphs
	tree := &TreeNode{
		Explain:    record,
		ChildNodes: []*TreeNode{},
	}
	for i := 1; i <= int(record.ChildrenNum); i++ {
		node := buildTree(graphs)
		tree.ChildNodes = append(tree.ChildNodes, node)
	}

	return tree
}

func traverse(tree *TreeNode, index int) {

	s := tree.Explain
	leveledList = append(leveledList, pterm.LeveledListItem{Level: index, Text: "Type: " + s.Type.String()})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: index, Text: "Alias: " + s.Alias})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: index, Text: "ChildrenNum: " + fmt.Sprint(s.ChildrenNum)})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: index, Text: "Uql: " + s.Uql})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: index, Text: "Infos: " + s.Infos})
	index++
	if tree == nil || tree.ChildNodes == nil || len(tree.ChildNodes) == 0 {
		return
	}
	for _, node := range tree.ChildNodes {
		traverse(node, index)
	}
}
