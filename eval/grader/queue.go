package grader

import (
	"context"
	"github.com/KiloProjects/kilonova"
	"github.com/google/btree"
	"log/slog"
	"sync"
)

// TODO: Currently unused

func subPriority(sub *kilonova.Submission) (priority int) {
	switch sub.Status {
	case kilonova.StatusWaiting:
		priority = 20
	case kilonova.StatusReevaling:
		priority = 50
	default:
		priority = 999
		slog.WarnContext(context.Background(), "Unexpected submission found to compute priority")
	}
	return
}

// Order first by priority, and then by ID
func entryCmp(a, b *kilonova.Submission) bool {
	if a == nil || b == nil {
		return false
	}
	if a.ID == b.ID {
		return false
	}
	return subPriority(a) < subPriority(b) || (subPriority(a) == subPriority(b) && a.ID < b.ID)
}

type evaluationQueue struct {
	treeMu sync.RWMutex
	tree   *btree.BTreeG[*kilonova.Submission]
}

func (q *evaluationQueue) Add(sub *kilonova.Submission) {
	q.treeMu.Lock()
	defer q.treeMu.Unlock()
	q.tree.ReplaceOrInsert(sub)
}

func (q *evaluationQueue) Has(sub *kilonova.Submission) bool {
	q.treeMu.RLock()
	defer q.treeMu.RUnlock()
	return q.tree.Has(sub)
}

func (q *evaluationQueue) Pop() *kilonova.Submission {
	q.treeMu.Lock()
	defer q.treeMu.Unlock()
	sub, ok := q.tree.DeleteMin()
	if !ok {
		return nil
	}
	return sub
}

func (q *evaluationQueue) Iterate() []*kilonova.Submission {
	q.treeMu.RLock()
	defer q.treeMu.RUnlock()
	var list []*kilonova.Submission
	q.tree.Ascend(func(item *kilonova.Submission) bool {
		list = append(list, item)
		return false
	})
	return list
}

func newEvaluationQueue() *evaluationQueue {
	return &evaluationQueue{tree: btree.NewG(2, entryCmp)}
}
