package judge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"gorm.io/gorm"
)

// Grader is the *actual* high-level grader
// Grader
type Grader struct {
	// These are the channels that are propagated to the box managers
	MasterTasks   chan common.Task
	MasterUpdater chan common.Updater
	DataManager   datamanager.Manager
	Managers      []*BoxManager

	ctx context.Context
	db  *gorm.DB
}

// NewGrader returns a new Grader instance (note that, as of the current architectural design, there should be only one grader)
func NewGrader(ctx context.Context, db *gorm.DB, dataManager datamanager.Manager) *Grader {
	taskChan := make(chan common.Task, 5)
	updateChan := make(chan common.Updater, 20)
	return &Grader{
		MasterTasks:   taskChan,
		MasterUpdater: updateChan,
		DataManager:   dataManager,
		db:            db,
		ctx:           ctx,
	}
}

// Start begins polling the DB for changes and sends them to the boxes
func (g *Grader) Start() {
	for _, mgr := range g.Managers {
		mgr.Start(g.ctx)
	}

	// DB Poller (pushes data to g.MasterTasks)
	go func() {
		// We don't want to use max CPU, so we poll every few seconds
		// TODO: DECREASE POLL TIME, THIS WAS SET HIGH FOR DEBUGGING PURPOSES
		ticker := time.NewTicker(4 * time.Second)
		for {
			select {
			case <-ticker.C:
				// poll db
				var tasks []common.Task
				g.db.Where("status = ?", common.StatusWaiting).
					Preload("Tests").Preload("Problem").Preload("Tests.Test").
					Find(&tasks)

				if len(tasks) > 0 {
					fmt.Printf("Found %d tasks\n", len(tasks))
				}

				// announce update
				for _, task := range tasks {
					g.MasterTasks <- task
				}
			case <-g.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	// DB Updater (fetches data from g.MasterUpdater)
	go func() {
		for {
			select {
			case update := <-g.MasterUpdater:
				if err := update.Update(g.db); err != nil {
					fmt.Println("GRADER DB UPDATE ERROR:", err)
				}
			case <-g.ctx.Done():
				return
			}
		}
	}()
}

// StopManagers does a graceful shutdown of all managers
func (g *Grader) StopManagers() error {
	type grErr struct {
		id  int
		err error
	}
	var errs []grErr
	for _, mgr := range g.Managers {
		if err := mgr.Cleanup(); err != nil {
			errs = append(errs, grErr{mgr.ID, err})
		}
	}
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return fmt.Errorf("Could not stop manager %d: %v", errs[0].id, errs[0].err)
	}
	retStr := "Multiple managers could not be stopped:\n"
	for _, err := range errs {
		retStr += fmt.Sprintf("Manager %d: %s\n", err.id, err.err)
	}

	return errors.New(retStr)
}

// Shutdown does a grateful shutdown of the grader
func (g *Grader) Shutdown() error {
	if err := g.StopManagers(); err != nil {
		return err
	}
	close(g.MasterTasks)
	close(g.MasterUpdater)
	return nil
}

// NewManager creates a new manager and assigns the master channels to it
func (g *Grader) NewManager(id int) error {
	mgr, err := NewBoxManager(id, g.DataManager, g.MasterTasks, g.MasterUpdater)
	if err != nil {
		return err
	}
	mgr.ToggleDebug()
	g.Managers = append(g.Managers, mgr)
	return nil
}
