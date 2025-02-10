package majortom

import (
	"errors"
	"github.com/paulmach/orb"
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
	rc := int64(math.Ceil(math.Pi * radius / float64(size)))
	ls := min((180.0 / float64(rc)), 89.0)
	mtg := MajorTomGrid{Size: size, Overlap: overlap, rowCount: rc, latSpacing: float64(ls)}
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
}

// rowLat calculates the latitude for a given row index.
func (g *MajorTomGrid) rowLat(rowIdx int64) float64 {
	return float64(-lonDeg) + float64(rowIdx)*g.latSpacing
}

// lonSpacing calculates the longitude spacing at a given latitude.
func (g *MajorTomGrid) lonSpacing(lat float64) float64 {

	latRad := toRad(min(max(lat, -89), 89))
	cir := 2 * math.Pi * radius * math.Cos(latRad)
	cols := math.Ceil(cir / float64(g.Size))
	return 360 / max(cols, 1)
}

// toRad converts degrees to radians.
func toRad(deg float64) float64 {
	return deg * math.Pi / latDeg
}

// GenerateGridCells divides the area of interest (AOI) into grid cells and returns those that intersect with the AOI.
func (g *MajorTomGrid) GenerateGridCells(geo orb.Geometry) ([]GridCell, error) {

	aoi := geo.Bound().ToPolygon()

	aoiMaxLon := aoi.Bound().Max.Lon()
	aoiMinLon := aoi.Bound().Min.Lon()
	if aoiMinLon > aoiMaxLon {
		aoiMaxLon += 360
	}
	minLat := aoi.Bound().Min.Lat()
	maxLat := aoi.Bound().Max.Lat()
	startRow := int64(math.Floor((minLat + lonDeg) / g.latSpacing))
	endRow := int64(math.Ceil((maxLat + lonDeg) / g.latSpacing))

	for g.rowLat(startRow) > minLat+lonDeg+1e-10 {
		startRow -= 1
	}
	for g.rowLat(endRow) < maxLat-1e-10 {
		endRow += 1
	}

	tiles := make([]GridCell, 0)
	mutex := &sync.Mutex{}

	halfLatSpacing := g.latSpacing / 2
	var wg sync.WaitGroup
	for rowIdx := startRow; rowIdx < endRow; rowIdx++ {
		wg.Add(1)
		go func(rowIdx int64) {
			defer wg.Done()
			lat := g.rowLat(rowIdx)
			lonSpacing := g.lonSpacing(lat)
			halfLonSpacing := lonSpacing / 2
			startCol := int(math.Floor((aoiMinLon + latDeg) / lonSpacing))
			endCol := int((aoiMaxLon + latDeg) / lonSpacing)

			for (-180.0 + float64(startCol)*lonSpacing) > (aoiMinLon + 1e-10) {
				startCol -= 1
			}
			for (-180.0 + float64(endCol)*lonSpacing) < (aoiMaxLon - 1e-10) {
				endCol += 1
			}

			for colIdx := startCol; colIdx < endCol; colIdx++ {
				lon := -latDeg + float64(colIdx)*lonSpacing
				p := orb.Polygon{{
					{lon, lat},
					{lon + lonSpacing, lat},
					{lon + lonSpacing, lat + g.latSpacing},
					{lon, lat + g.latSpacing},
					{lon, lat}}}
				mutex.Lock()
				tiles = append(tiles, GridCell{p})
				mutex.Unlock()

				if g.Overlap {
					overlapLon := lon + halfLonSpacing
					overlapLat := lat + halfLatSpacing
					eastOverlapCell := orb.Polygon{{
						{overlapLon, overlapLat},
						{overlapLon + lonSpacing, overlapLat},
						{overlapLon + lonSpacing, overlapLat + g.latSpacing},
						{overlapLon, overlapLat + g.latSpacing},
						{overlapLon, overlapLat},
					}}

					if eastOverlapCell.Bound().Intersects(aoi.Bound()) {
						mutex.Lock()
						tiles = append(tiles, GridCell{eastOverlapCell})
						mutex.Unlock()
					}

					//southOverlapCell := orb.Polygon{{
					//	{lon, lat - halfLatSpacing},
					//	{lon + lonSpacing, lat - halfLatSpacing},
					//	{lon + lonSpacing, lat + g.latSpacing - halfLatSpacing},
					//	{lon, lat + g.latSpacing - halfLatSpacing},
					//	{lon, lat - halfLatSpacing},
					//}}
					//if southOverlapCell.Bound().Intersects(aoi.Bound()) {
					//	mutex.Lock()
					//	tiles = append(tiles, GridCell{southOverlapCell})
					//	mutex.Unlock()
					//}

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
		searchId = id[0:geohashPrecision]
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

	cells, err := g.GenerateGridCells(&p)
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		if cell.Id() == searchId {
			return &cell, nil
		}
	}
	//expand bbox by 10% if nothing is found
	p = expandBound(b, 10).ToPolygon()
	cells, err = g.GenerateGridCells(p)
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		if cell.Id() == searchId {
			return &cell, nil
		}
	}
	return nil, errors.New("cell not found")
}

func expandBound(bound orb.Bound, percentage float64) orb.Bound {
	// Calculate the width and height of the bounding box.
	width := bound.Max[0] - bound.Min[0]
	height := bound.Max[1] - bound.Min[1]

	// Calculate the expansion amounts for each dimension.
	expandX := width * (percentage / 100.0)
	expandY := height * (percentage / 100.0)

	// Create a new expanded bounding box.
	newBound := orb.Bound{
		Min: orb.Point{bound.Min[0] - expandX/2, bound.Min[1] - expandY/2},
		Max: orb.Point{bound.Max[0] + expandX/2, bound.Max[1] + expandY/2},
	}

	return newBound
}
