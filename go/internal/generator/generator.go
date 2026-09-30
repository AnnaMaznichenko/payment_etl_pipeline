package generator

import (
	"context"
	"log"
	"sync"
	"time"

	"payment_etl_pipeline/internal/interfaces"
)

type generator struct {
	sources []interfaces.SourceGenerator
}

func NewGenerator(sources []interfaces.SourceGenerator) interfaces.Generator {
	return &generator{
		sources: sources,
	}
}

func (g *generator) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, s := range g.sources {
		wg.Add(1)
		go func(sg interfaces.SourceGenerator) {
			defer wg.Done()
			g.runSource(ctx, sg)
		}(s)
	}
	wg.Wait()
	return nil
}

func (g *generator) runSource(ctx context.Context, sg interfaces.SourceGenerator) {
	ticker := time.NewTicker(sg.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sg.Generate(ctx); err != nil {
				log.Printf("generator %s error: %v", sg.Name(), err)
			}
		}
	}
}
