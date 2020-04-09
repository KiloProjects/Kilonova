package eval

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/jinzhu/gorm"
)

// StartEvalListener starts listening for all tasks
func StartEvalListener(ctx context.Context, db *gorm.DB, config *models.Config) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case <-ticker.C:
				var newToEval []models.EvalTest
				db.Find(&newToEval)
				if len(newToEval) == 0 {
					continue
				}
				for _, toEval := range newToEval {
					fmt.Println(toEval)
					toEval.Test.Score = 100
				}
				db.Save(newToEval)
			}
		}
	}()
}
