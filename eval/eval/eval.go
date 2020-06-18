package eval

import (
	"github.com/AlexVasiluta/kilonova/eval/box"
	"github.com/AlexVasiluta/kilonova/models"
)

// BoxManager manages a box with eval-based tasks
type BoxManager struct {
	Box      *box.Box
	taskChan chan models.Task
}

// BoxOrchestrator manages multiple box managers
type BoxOrchestrator struct {
	Boxes []BoxManager
}

// Cleanup cleans up the boxes
func (b *BoxManager) Cleanup() error {
	return b.Box.Cleanup()
}

// NewBoxManager creates a new box
func NewBoxManager(numBoxes int) *BoxManager {
	bm := &BoxManager{Box: box.NewBox(box.Config{})}
	return bm
}
