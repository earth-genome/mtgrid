# mtgrid

[![Go CI](https://github.com/earth-genome/mtgrid/actions/workflows/smoke.yaml/badge.svg)](https://github.com/earth-genome/mtgrid/actions/workflows/smoke.yaml)

This is an implementation of the ESA [Major TOM](https://github.com/ESA-PhiLab/Major-TOM) equal area grid in Go.  
This code is based on work in the ESA-PhiLab repository, specifically [here](https://github.com/ESA-PhiLab/Major-TOM/blob/main/src/grid.py)

## Requirements

- Go 1.26.0 or later
- Module path: `github.com/earth-genome/mtgrid` (import the `majortom` subpackage for the public API)

## Usage

Also see [tests](majortom/mtgrid_test.go) and the [API reference](docs/api.md).

```go
package main

import (
	"fmt"

	"github.com/earth-genome/mtgrid/majortom"
	"github.com/paulmach/orb"
)

func main() {
	// Create a simple polygon; this is a 1/10th-of-a-degree square.
	p := orb.Polygon{{{0.0, 0.0}, {0.0, 0.1}, {0.1, 0.1}, {0.1, 0.0}, {0.0, 0.0}}}

	// Instantiate a grid with 320 m cells and overlap enabled.
	grid := majortom.NewGrid(320, true)

	// Generate cells that intersect the area of interest.
	cells, _ := grid.GenerateGridCells(&p)
	for _, cell := range cells {
		fmt.Printf("cell id %s\n", cell.Id())
	}

	// Look up a cell by geohash ID (longer IDs are truncated to 11 characters).
	cell, _ := grid.CellFromId("dr19n8ggbf9")

	// Map a prior-grid cell ID to the current primary cell.
	migrated, _ := grid.MigrateCellId("dr18zj1ntew")

	_, _ = cell, migrated
}
```

### Public API

| Method | Description |
|--------|-------------|
| `NewGrid(size, overlap)` | Create a grid. `size` is the nominal cell edge length in **meters**; `overlap` enables half-cell-offset overlap cells. |
| `GenerateGridCells(geo)` | Return all grid cells that intersect an `orb.Geometry` (polygon, multipolygon, etc.). |
| `CellFromId(id)` | Look up a cell by geohash ID. IDs longer than 11 characters are truncated before lookup. |
| `MigrateCellId(oldId)` | Map a cell ID from a prior grid version to the current grid's primary cell. |

Each `GridCell` exposes an `Id()` method that returns an **11-character geohash** derived from the cell centroid. When `overlap` is `true`, additional cells offset by half the lat/lon spacing are generated alongside the primary grid.

Before:
![Before](imgs/before.png)

After:
![After](./imgs/after.png)
