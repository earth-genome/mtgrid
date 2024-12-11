package majortom

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/pierrre/geohash"
	"math"
	"sync"
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
}

// Id returns a geohash string that uniquely identifies the GridCell.
func (gc *GridCell) Id() string {
	return geohash.Encode(gc.Bound().Center().Lat(), gc.Bound().Center().Lon(), geohashPrecision)
}

// NewGrid creates a new MajorTomGrid with the specified size and overlap settings.
func NewGrid(size uint64, overlap bool) *MajorTomGrid {
	return &MajorTomGrid{Size: size, Overlap: overlap}
}

// MajorTomGrid represents a geographical grid with specified size and overlap properties.
type MajorTomGrid struct {
	Size    uint64
	Overlap bool
	Width   int64
	Height  int64
}

// latSpacing calculates the latitude spacing between rows based on the number of rows.
func (g *MajorTomGrid) latSpacing(rows int64) float64 {
	return latDeg / float64(rows)
}

// rowCount calculates the number of rows in the grid based on the grid size.
func (g *MajorTomGrid) rowCount() int64 {
	return int64(math.Ceil(math.Pi * radius / float64(g.Size)))
}

// rowLat calculates the latitude for a given row index.
func (g *MajorTomGrid) rowLat(rowIdx int64) float64 {
	return float64(-lonDeg) + float64(rowIdx)*g.latSpacing(g.rowCount())
}

// lonSpacing calculates the longitude spacing at a given latitude.
func (g *MajorTomGrid) lonSpacing(lat float64) float64 {

	latRad := toRad(lat)
	cir := 2 * math.Pi * radius * math.Cos(latRad)
	cols := math.Ceil(cir / float64(g.Size))
	return 360 / cols
}

// toRad converts degrees to radians.
func toRad(deg float64) float64 {
	return deg * math.Pi / latDeg
}

// GenerateGridCells divides the area of interest (AOI) into grid cells and returns those that intersect with the AOI.
func (g *MajorTomGrid) GenerateGridCells(geo orb.Geometry) ([]GridCell, error) {

	aoi := geo.Bound().ToPolygon()
	rows := g.rowCount()
	startRow := max(int64(0), int64((aoi.Bound().Min.Lat()+lonDeg)/g.latSpacing(rows)))
	endRow := min(rows, int64((aoi.Bound().Max.Lat()+lonDeg)/g.latSpacing(rows))+1)
	aoiMaxLon := aoi.Bound().Max.Lon()
	aoiMinLon := aoi.Bound().Min.Lon()

	tiles := make([]GridCell, 0)
	mutex := &sync.Mutex{}

	latSpacing := g.latSpacing(rows)
	halfLatSpacing := latSpacing / 2
	var wg sync.WaitGroup
	for rowIdx := startRow; rowIdx < endRow; rowIdx++ {
		wg.Add(1)
		go func(rowIdx int64) {
			defer wg.Done()
			lat := g.rowLat(rowIdx)
			lonSpacing := g.lonSpacing(lat)
			halfLonSpacing := lonSpacing / 2
			startCol := max(0, int((aoiMinLon+latDeg)/lonSpacing))
			endCol := min(int(360/lonSpacing), int((aoiMaxLon+latDeg)/lonSpacing)+1)

			for colIdx := startCol; colIdx < endCol; colIdx++ {
				lon := -latDeg + float64(colIdx)*lonSpacing
				p := orb.Polygon{{
					{lon, lat},
					{lon + lonSpacing, lat},
					{lon + lonSpacing, lat + latSpacing},
					{lon, lat + latSpacing},
					{lon, lat}}}
				mutex.Lock()
				tiles = append(tiles, GridCell{p})
				mutex.Unlock()

				if g.Overlap {
					eastOverlapCell := orb.Polygon{{
						{lon + halfLonSpacing, lat},
						{lon + lonSpacing + halfLonSpacing, lat},
						{lon + lonSpacing + halfLonSpacing, lat + latSpacing},
						{lon + halfLonSpacing, lat + latSpacing},
						{lon + halfLonSpacing, lat},
					}}

					if eastOverlapCell.Bound().Intersects(aoi.Bound()) {
						mutex.Lock()
						tiles = append(tiles, GridCell{eastOverlapCell})
						mutex.Unlock()
					}

					southOverlapCell := orb.Polygon{{
						{lon, lat - halfLatSpacing},
						{lon + lonSpacing, lat - halfLatSpacing},
						{lon + lonSpacing, lat + latSpacing - halfLatSpacing},
						{lon, lat + latSpacing - halfLatSpacing},
						{lon, lat - halfLatSpacing},
					}}
					if southOverlapCell.Bound().Intersects(aoi.Bound()) {
						mutex.Lock()
						tiles = append(tiles, GridCell{southOverlapCell})
						mutex.Unlock()
					}

				}
			}
		}(rowIdx)
	}
	wg.Wait()
	return tiles, nil
}

// CellFromId retrieves a GridCell from its geohash ID.
func (g *MajorTomGrid) CellFromId(id string) (*GridCell, error) {

	searchId := id
	if len(id) > geohashPrecision {
		searchId = id[0 : geohashPrecision-1]
	}

	box, err := geohash.Decode(searchId)
	if err != nil {
		return nil, err
	}
	b := orb.Bound{
		Min: orb.Point{box.Lon.Min, box.Lat.Min},
		Max: orb.Point{box.Lon.Max, box.Lat.Max},
	}
	p := b.ToPolygon()
	centroid := b.Center()
	cells, err := g.GenerateGridCells(&p)
	if err != nil {
		return nil, err
	}
	leastDist := math.MaxFloat64
	var closestCell *GridCell
	for _, cell := range cells {
		if cell.Id() == id {
			return &cell, nil
		} else {
			dist := planar.Distance(cell.Bound().Center(), centroid)
			if dist < leastDist {
				leastDist = dist
				closestCell = &cell
			}
		}
	}
	return closestCell, nil
}
