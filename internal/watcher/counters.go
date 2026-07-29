package watcher

import "github.com/jonyd/gobsidian/internal/service"

// Stats devolve os contadores atuais do watcher para o vault_stats.
func (w *Watcher) Stats() service.WatchCounters {
	return service.WatchCounters{
		Active:          true,
		EventsReceived:  w.received.Load(),
		EventsDropped:   w.dropped.Load(),
		EventsProcessed: w.processed.Load(),
		EventsSkipped:   w.skipped.Load(),
		Reconciliations: w.reconciliations.Load(),
	}
}
