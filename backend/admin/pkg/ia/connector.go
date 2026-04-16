package ia

import "context"

// IAConnector est l'interface que tout connecteur IA doit implémenter.
type IAConnector interface {
	Analyze(ctx context.Context, prompt string) (string, error)
}
