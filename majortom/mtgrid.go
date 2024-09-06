package majortom

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/pierrre/geohash"
	"math"
	"math/big"
	"sync"
)

const (
	RADIUS = 6378137
	LatDeg = 180.0
	LonDeg = 90.0
)

type GridCell struct {
	orb.Polygon
}

func (gc *GridCell) Id() string {
	return geohash.Encode(gc.Bound().Center().Lat(), gc.Bound().Center().Lon(), 20)
}

type Grid struct {
	Size    int64
	Overlap bool
	Width   int64
	Height  int64
}

func (g *Grid) latSpacing(rows int64) float64 {
	return LatDeg / float64(rows)
}
func (g *Grid) rowCount() int64 {
	return int64(math.Ceil(math.Pi * RADIUS / float64(g.Size)))
}
func (g *Grid) rowLat(rowIdx int64) float64 {
	return float64(-LonDeg) + float64(rowIdx)*g.latSpacing(g.rowCount())
}
func (g *Grid) lonSpacing(lat float64) float64 {

	latRad := toRad(lat)
	cir := 2 * math.Pi * RADIUS * math.Cos(latRad)
	cols := math.Ceil(cir / float64(g.Size))
	return 360 / cols
}

func toRad(deg float64) float64 {
	return deg * math.Pi / LatDeg
}

func New(size int64, overlap bool) *Grid {
	return &Grid{Size: size, Overlap: overlap}
}

func (g *Grid) TilePolygon(aoi *orb.Polygon) ([]GridCell, error) {

	rows := g.rowCount()
	startRow := max(int64(0), int64((aoi.Bound().Min.Lat()+LonDeg)/g.latSpacing(rows)))
	endRow := min(rows, int64((aoi.Bound().Max.Lat()+LonDeg)/g.latSpacing(rows))+1)

	tiles := make([]GridCell, 0)
	mutex := &sync.Mutex{}

	latSpacing := g.latSpacing(rows)
	halfLatSpacing := latSpacing / 2
	var wg sync.WaitGroup
	for rowIdx := startRow; rowIdx <= endRow; rowIdx++ {
		wg.Add(1)
		go func(rowIdx int64) {
			defer wg.Done()
			lat := g.rowLat(rowIdx)
			lonSpacing := g.lonSpacing(lat)
			halfLonSpacing := lonSpacing / 2
			startCol := max(0, int((aoi.Bound().Min.Lon()+LatDeg)/lonSpacing))
			endCol := min(int(360/lonSpacing), int((aoi.Bound().Max.Lon()+LatDeg)/lonSpacing)+1)

			for colIdx := startCol; colIdx <= endCol; colIdx++ {
				lon := -LatDeg + float64(colIdx)*lonSpacing
				p := orb.Polygon{{
					{lon, lat},
					{lon + lonSpacing, lat},
					{lon + lonSpacing, lat + latSpacing},
					{lon, lat + latSpacing},
					{lon, lat}}}
				if p.Bound().Intersects(aoi.Bound()) {
					mutex.Lock()
					tiles = append(tiles, GridCell{p})
					mutex.Unlock()
				}
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

// TilePolygonToGrid bahves like TilePolygon
func (g *Grid) TilePolygonToChan(aoi *orb.MultiPolygon, geochan chan GridCell) {

	rows := g.rowCount()
	startRow := max(int64(0), int64((aoi.Bound().Min.Lat()+LonDeg)/g.latSpacing(rows)))
	endRow := min(rows, int64((aoi.Bound().Max.Lat()+LonDeg)/g.latSpacing(rows))+1)

	for rowIdx := int64(startRow); rowIdx <= endRow; rowIdx++ {
		lat := g.rowLat(rowIdx)
		lonSpacing := g.lonSpacing(lat)
		latSpacing := g.latSpacing(rows)
		halfLatSpacing := g.latSpacing(rows) / 2
		halfLonSpacing := lonSpacing / 2

		startCol := max(0, int((aoi.Bound().Min.Lon()+LatDeg)/lonSpacing))
		endCol := min(int(360/lonSpacing), int((aoi.Bound().Max.Lon()+LatDeg)/lonSpacing)+1)

		for colIdx := startCol; colIdx <= endCol; colIdx++ {
			lon := -LatDeg + float64(colIdx)*lonSpacing
			p := orb.Polygon{{
				{lon, lat},
				{lon + lonSpacing, lat},
				{lon + lonSpacing, lat + latSpacing},
				{lon, lat + latSpacing},
				{lon, lat}}}
			if p.Bound().Intersects(aoi.Bound()) {
				geochan <- GridCell{p}
			}
			eastOverlapCell := orb.Polygon{{
				{lon + halfLonSpacing, lat},
				{lon + lonSpacing + halfLonSpacing, lat},
				{lon + lonSpacing + halfLonSpacing, lat + latSpacing},
				{lon + halfLonSpacing, lat + latSpacing},
				{lon + halfLonSpacing, lat},
			}}
			if eastOverlapCell.Bound().Intersects(aoi.Bound()) {
				geochan <- GridCell{eastOverlapCell}
			}
			southOverlapCell := orb.Polygon{{
				{lon, lat - halfLatSpacing},
				{lon + lonSpacing, lat - halfLatSpacing},
				{lon + lonSpacing, lat + latSpacing - halfLatSpacing},
				{lon, lat + latSpacing - halfLatSpacing},
				{lon, lat - halfLatSpacing},
			}}
			if southOverlapCell.Bound().Intersects(aoi.Bound()) {
				geochan <- GridCell{southOverlapCell}
			}
		}
	}
	//indicate we're done
	close(geochan)
}

func (g *Grid) CountCells(aoi *orb.Polygon) *big.Int {

	rows := g.rowCount()
	startRow := max(int64(0), int64((aoi.Bound().Min.Lat()+LonDeg)/g.latSpacing(rows)))
	endRow := min(rows, int64((aoi.Bound().Max.Lat()+LonDeg)/g.latSpacing(rows))+1)

	tiles := big.NewInt(0)
	biggestRow := int64(0)
	for rowIdx := int64(startRow); rowIdx <= endRow; rowIdx++ {
		lat := g.rowLat(rowIdx)
		lonSpacing := g.lonSpacing(lat)

		endCol := min(int(360/lonSpacing), int((aoi.Bound().Max.Lon()+LatDeg)/lonSpacing)+1)
		tiles = tiles.Add(big.NewInt(int64(endCol)), tiles)
		if int64(endCol) > biggestRow {
			biggestRow = int64(endCol)
		}

	}

	return tiles
}

func (g *Grid) CellFromId(id string) (*GridCell, error) {

	searchId := id
	if len(id) == 20 {
		searchId = id[0:18]
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
	cells, err := g.TilePolygon(&p)
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
