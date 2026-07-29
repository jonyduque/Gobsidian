package watcher

// Counters e o retrato dos contadores do watcher. Mora aqui, e nao em
// internal/service: as camadas abaixo do servico nao o conhecem
// (ARCHITECTURE.md §1). Quem casa os dois lados e o adaptador em
// cmd/gobsidian/serve.go.
type Counters struct {
	Active            bool
	EventsReceived    int64
	EventsDropped     int64            // soma de DroppedByReason
	DroppedByReason   map[string]int64 // chmod, outside_vault, excluded, unknown_op
	EventsCoalesced   int64
	EventsProcessed   int64
	EventsSkipped     int64
	Reconciliations   int64
	ReconciledUpdated int64
	ReconciledRemoved int64
}

// Stats devolve os contadores atuais do watcher.
func (w *Watcher) Stats() Counters {
	porMotivo := map[string]int64{
		"chmod":         w.droppedChmod.Load(),
		"outside_vault": w.droppedOutsideVault.Load(),
		"excluded":      w.droppedExcluded.Load(),
		"unknown_op":    w.droppedUnknownOp.Load(),
	}
	var soma int64
	for _, n := range porMotivo {
		soma += n
	}
	return Counters{
		Active:            w.active.Load(),
		EventsReceived:    w.received.Load(),
		EventsDropped:     soma,
		DroppedByReason:   porMotivo,
		EventsCoalesced:   w.coalesced.Load(),
		EventsProcessed:   w.processed.Load(),
		EventsSkipped:     w.skipped.Load(),
		Reconciliations:   w.reconciliations.Load(),
		ReconciledUpdated: w.reconciledUpdated.Load(),
		ReconciledRemoved: w.reconciledRemoved.Load(),
	}
}
