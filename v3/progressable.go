package pb

import (
	"log"
	"time"
)

type Progressable interface {
	Total() int64
	Value() int64
	FinishedChan() <-chan struct{}
}

func RegisterProgressable(pr Progressable, removeFunc func(*ProgressBar)) *ProgressBar {
	pb := new(ProgressBar)
	go progressWorker(pr, pb, removeFunc)
	return pb
}

func progressWorker(pr Progressable, pb *ProgressBar, removeFunc func(*ProgressBar)) {
	fc := pr.FinishedChan()
	ticker := time.NewTicker(time.Second)
	defer pb.Finish()
	defer removeFunc(pb)
	for {
		select {
		case <-ticker.C:
			_ = pb.SetTotal(pr.Total())
			_ = pb.SetCurrent(pr.Value())
			log.Println("Updating bar", pr.Total(), pr.Value())
		case _, ok := <-fc:
			if !ok {
				return
			}
		}
	}
}
