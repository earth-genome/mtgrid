package majortom

import (
	"errors"
	"math"
	"sync"

	"github.com/paulmach/orb"
	"github.com/pierrre/geohash"
	predicates "github.com/tingold/orb-predicates"
)

const (
	radius           = 6378137
	latDeg           = 180.0
	lonDeg           = 90.0
	geohashPrecision = 11
)

// GridCell represents a cell in the grid, defined by an orb.Polygon.
type GridCell struct {
	orb.Polygon
	id string
}

// newGridCell creates a GridCell and pre-computes its geohash ID.
func newGridCell(p orb.Polygon) GridCell {
	c := p.Bound().Center()
	return GridCell{
		Polygon: p,
		id:      geohash.Encode(c.Lat(), c.Lon(), geohashPrecision),
	}
}

// Id returns a geohash string that uniquely identifies the GridCell.
func (gc *GridCell) Id() string {
	return gc.id
}

// NewGrid creates a new MajorTomGrid with the specified size and overlap settings.
func NewGrid(size uint64, overlap bool) *MajorTomGrid {
	rc := int64(math.Max(2, math.Ceil(math.Pi*radius/float64(size))))
	ls := min((180.0 / float64(rc)), 89.0)
	var latOff float64
	if rc%2 != 0 {
		latOff = ls / 2
	}
	mtg := MajorTomGrid{Size: size, Overlap: overlap, rowCount: rc, latSpacing: float64(ls), latOffset: latOff}
	return &mtg
}

// MajorTomGrid represents a geographical grid with specified size and overlap properties.
type MajorTomGrid struct {
	Size       uint64
	Overlap    bool
	Width      int64
	Height     int64
	rowCount   int64
	latSpacing float64
	latOffset  float64
}

// rowLat calculates the latitude for a given row index.
func (g *MajorTomGrid) rowLat(rowIdx int64) float64 {
	return float64(-lonDeg) + g.latOffset + float64(rowIdx)*g.latSpacing
}

// lonSpacing calculates the longitude spacing at a given latitude.
func (g *MajorTomGrid) lonSpacing(lat float64) float64 {
	latRad := toRad(min(max(lat, -89), 89))
	cir := 2 * math.Pi * radius * math.Cos(latRad)
	cols := math.Ceil(cir / float64(g.Size))
	return 360 / max(cols, 1)
}

// lonOffset returns the longitude centering offset for a given longitude spacing.
// When the number of columns is odd, columns are shifted by half a spacing so
// that the prime meridian falls on a grid line.
func (g *MajorTomGrid) lonOffset(lonSpacing float64) float64 {
	var nCols int
	if lonSpacing > 0 {
		nCols = int(math.Round(360 / lonSpacing))
	}
	if nCols%2 != 0 {
		return lonSpacing / 2
	}
	return 0
}

// colLon calculates the longitude for a given column index.
func (g *MajorTomGrid) colLon(colIdx int, lonSpacing, lonOff float64) float64 {
	return -latDeg + lonOff + float64(colIdx)*lonSpacing
}

// toRad converts degrees to radians.
func toRad(deg float64) float64 {
	return deg * math.Pi / latDeg
}

// derefGeometry returns the underlying value type for pointer geometry types.
// The orb-predicates library's type switches only match value types (e.g. orb.Polygon),
// not pointer types (e.g. *orb.Polygon), so we must unwrap them.
func derefGeometry(g orb.Geometry) orb.Geometry {
	switch v := g.(type) {
	case *orb.Point:
		if v == nil {
			return nil
		}
		return *v
	case *orb.MultiPoint:
		if v == nil {
			return nil
		}
		return *v
	case *orb.LineString:
		if v == nil {
			return nil
		}
		return *v
	case *orb.MultiLineString:
		if v == nil {
			return nil
		}
		return *v
	case *orb.Ring:
		if v == nil {
			return nil
		}
		return *v
	case *orb.Polygon:
		if v == nil {
			return nil
		}
		return *v
	case *orb.MultiPolygon:
		if v == nil {
			return nil
		}
		return *v
	case *orb.Collection:
		if v == nil {
			return nil
		}
		return *v
	case *orb.Bound:
		if v == nil {
			return nil
		}
		return *v
	default:
		return g
	}
}

// GenerateGridCells divides the area of interest (AOI) into grid cells and returns those that intersect with the AOI.
func (g *MajorTomGrid) GenerateGridCells(geo orb.Geometry) ([]GridCell, error) {

	// Ensure we have a value type for the predicates library.
	geo = derefGeometry(geo)
	if geo == nil {
		return nil, errors.New("geometry must not be nil")
	}

	// Use the bounding box only to determine the row/column search space.
	aoiBound := geo.Bound()

	aoiMaxLon := aoiBound.Max.Lon()
	aoiMinLon := aoiBound.Min.Lon()
	if aoiMinLon > aoiMaxLon {
		aoiMaxLon += 360
	}
	aoiMinLat := aoiBound.Min.Lat()
	aoiMaxLat := aoiBound.Max.Lat()
	startRow := int64(math.Floor((aoiMinLat + lonDeg - g.latOffset) / g.latSpacing))
	endRow := int64(math.Ceil((aoiMaxLat + lonDeg - g.latOffset) / g.latSpacing))

	for g.rowLat(startRow) > aoiMinLat+1e-10 {
		startRow -= 1
	}
	for g.rowLat(endRow) < aoiMaxLat-1e-10 {
		endRow += 1
	}

	estimatedCap := int(endRow-startRow) * 2
	tiles := make([]GridCell, 0, estimatedCap)
	mutex := &sync.Mutex{}

	halfLatSpacing := g.latSpacing / 2

	var wg sync.WaitGroup
	for rowIdx := startRow; rowIdx < endRow; rowIdx++ {
		wg.Add(1)
		go func(rowIdx int64) {
			defer wg.Done()
			lat := g.rowLat(rowIdx)
			lonSpacing := g.lonSpacing(lat)
			lonOff := g.lonOffset(lonSpacing)
			halfLonSpacing := lonSpacing / 2
			startCol := int(math.Floor((aoiMinLon + latDeg - lonOff) / lonSpacing))
			endCol := int(math.Ceil((aoiMaxLon + latDeg - lonOff) / lonSpacing))

			for g.colLon(startCol, lonSpacing, lonOff) > (aoiMinLon + 1e-10) {
				startCol -= 1
			}
			for g.colLon(endCol, lonSpacing, lonOff) < (aoiMaxLon - 1e-10) {
				endCol += 1
			}

			// Collect results locally to reduce mutex contention.
			var localTiles []GridCell

			for colIdx := startCol; colIdx < endCol; colIdx++ {
				lon := g.colLon(colIdx, lonSpacing, lonOff)
				cellMaxLon := lon + lonSpacing
				cellMaxLat := lat + g.latSpacing

				// Inline bounding-box pre-check to avoid predicate dispatch overhead.
				if cellMaxLon < aoiMinLon || lon > aoiMaxLon ||
					cellMaxLat < aoiMinLat || lat > aoiMaxLat {
					continue
				}

				p := orb.Polygon{{
					{lon, lat},
					{cellMaxLon, lat},
					{cellMaxLon, cellMaxLat},
					{lon, cellMaxLat},
					{lon, lat}}}

				// Use precise geometric intersection.
				if predicates.Intersects(p, geo) {
					localTiles = append(localTiles, newGridCell(p))
				}

				if g.Overlap {
					overlapLon := lon + halfLonSpacing
					overlapLat := lat + halfLatSpacing
					overlapMaxLon := overlapLon + lonSpacing
					overlapMaxLat := overlapLat + g.latSpacing

					// Inline bounding-box pre-check for overlap cell.
					if overlapMaxLon < aoiMinLon || overlapLon > aoiMaxLon ||
						overlapMaxLat < aoiMinLat || overlapLat > aoiMaxLat {
						continue
					}

					eastOverlapCell := orb.Polygon{{
						{overlapLon, overlapLat},
						{overlapMaxLon, overlapLat},
						{overlapMaxLon, overlapMaxLat},
						{overlapLon, overlapMaxLat},
						{overlapLon, overlapLat},
					}}

					if predicates.Intersects(eastOverlapCell, geo) {
						localTiles = append(localTiles, newGridCell(eastOverlapCell))
					}
				}
			}

			// Append all row results under a single lock.
			if len(localTiles) > 0 {
				mutex.Lock()
				tiles = append(tiles, localTiles...)
				mutex.Unlock()
			}
		}(rowIdx)
	}
	wg.Wait()
	return tiles, nil
}

// CellFromId retrieves a GridCell from its geohash ID.
// It computes the row/column index directly from the geohash center coordinates
// rather than generating and searching all cells in the area.
func (g *MajorTomGrid) CellFromId(id string) (*GridCell, error) {

	searchId := id
	if len(id) > geohashPrecision {
		searchId = id[:geohashPrecision]
	}

	box, err := geohash.Decode(searchId)
	if err != nil {
		return nil, err
	}
	centerLat := box.Lat.Mid()
	centerLon := box.Lon.Mid()

	// Try the direct row/col computation and nearby neighbors to handle
	// floating-point edge cases and overlap cells.
	halfLatSpacing := g.latSpacing / 2
	for _, rowOffset := range []int64{0, -1, 1} {
		rowIdx := int64(math.Floor((centerLat+lonDeg-g.latOffset)/g.latSpacing)) + rowOffset
		rowLat := g.rowLat(rowIdx)
		lonSpacing := g.lonSpacing(rowLat)
		lonOff := g.lonOffset(lonSpacing)
		halfLonSpacing := lonSpacing / 2

		for _, colOffset := range []int{0, -1, 1} {
			colIdx := int(math.Floor((centerLon+latDeg-lonOff)/lonSpacing)) + colOffset
			cellLon := g.colLon(colIdx, lonSpacing, lonOff)

			// Try the regular cell at this row/col.
			p := orb.Polygon{{
				{cellLon, rowLat},
				{cellLon + lonSpacing, rowLat},
				{cellLon + lonSpacing, rowLat + g.latSpacing},
				{cellLon, rowLat + g.latSpacing},
				{cellLon, rowLat},
			}}
			cell := newGridCell(p)
			if cell.Id() == searchId {
				return &cell, nil
			}

			// Try the overlap cell offset by half spacing.
			if g.Overlap {
				overlapLon := cellLon + halfLonSpacing
				overlapLat := rowLat + halfLatSpacing
				op := orb.Polygon{{
					{overlapLon, overlapLat},
					{overlapLon + lonSpacing, overlapLat},
					{overlapLon + lonSpacing, overlapLat + g.latSpacing},
					{overlapLon, overlapLat + g.latSpacing},
					{overlapLon, overlapLat},
				}}
				oCell := newGridCell(op)
				if oCell.Id() == searchId {
					return &oCell, nil
				}
			}
		}
	}

	return nil, errors.New("cell not found")
}

// MigrateCellId maps a cell ID from a prior grid version to the current grid.
// It decodes the geohash to recover the approximate centroid, then returns
// the current-grid primary cell that contains that point.
func (g *MajorTomGrid) MigrateCellId(oldId string) (*GridCell, error) {
	searchId := oldId
	if len(oldId) > geohashPrecision {
		searchId = oldId[:geohashPrecision]
	}
	if len(searchId) != geohashPrecision {
		return nil, errors.New("cell ID must be at least 11 characters")
	}

	box, err := geohash.Decode(searchId)
	if err != nil {
		return nil, err
	}
	lat := box.Lat.Mid()
	lon := box.Lon.Mid()

	rowIdx := int64(math.Floor((lat + lonDeg - g.latOffset) / g.latSpacing))
	rowLat := g.rowLat(rowIdx)
	lonSpacing := g.lonSpacing(rowLat)
	lonOff := g.lonOffset(lonSpacing)
	colIdx := int(math.Floor((lon + latDeg - lonOff) / lonSpacing))
	cellLon := g.colLon(colIdx, lonSpacing, lonOff)

	p := orb.Polygon{{
		{cellLon, rowLat},
		{cellLon + lonSpacing, rowLat},
		{cellLon + lonSpacing, rowLat + g.latSpacing},
		{cellLon, rowLat + g.latSpacing},
		{cellLon, rowLat},
	}}
	return new(newGridCell(p)), nil
}
