package mtgrid

import "math"
import "github.com/paulmach/orb"

const (
	RADIUS = 6378137
	LatDeg = 180.0
	LonDeg = 90
)

type Grid struct {
	Size    int64
	Overlap bool
}

func (g *Grid) latSpacing(rows int64) float64 {
	return LatDeg / float64(rows)
}
func (g *Grid) rowCount() int64 {
	return int64(math.Ceil(math.Pi * RADIUS / float64(g.Size)))
}
func (g *Grid) rowLat(rowIdx int64) float64 {
	return float64(-90) + float64(rowIdx)*g.latSpacing(g.rowCount())
}

func (g *Grid) lonSpacing(lat float64) float64 {

	latRad := toRad(lat)
	cir := 2 * math.Pi * RADIUS * math.Cos(latRad)
	cols := math.Ceil(cir / float64(g.Size))
	return 360 / cols
}

func (g *Grid) TilePolygon(aoi *orb.Polygon) ([]orb.Polygon, error) {

	rows := g.rowCount()
	startRow := max(int64(0), int64((aoi.Bound().Min.Lat()+90)/g.latSpacing(rows)))
	endRow := min(rows, int64((aoi.Bound().Max.Lat()+90)/g.latSpacing(rows))+1)

	tiles := make([]orb.Polygon, 0)

	for rowIdx := int64(startRow); rowIdx <= endRow; rowIdx++ {
		lat := g.rowLat(rowIdx)
		lonSpacing := g.lonSpacing(lat)
		//halfLatSpacing := g.latSpacing(rows) / 2
		//halfLonSpacing := lonSpacing / 2

		startCol := max(0, int((aoi.Bound().Min.Lon()+180)/lonSpacing))
		endCol := min(int(360/lonSpacing), int((aoi.Bound().Max.Lon()+180)/lonSpacing)+1)

		for colIdx := startCol; colIdx <= endCol; colIdx++ {

			lon := -180 + float64(colIdx)*lonSpacing
			p := orb.Polygon{{{lon, lat},
				{lon + lonSpacing, lat},
				{lon + lonSpacing, lat + g.latSpacing(rows)},
				{lon, lat + g.latSpacing(rows)},
				{lon, lat}}}
			if p.Bound().Intersects(aoi.Bound()) {
				tiles = append(tiles, p)
			}
		}
	}
	return tiles, nil
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}
func New(size int64, overlap bool) *Grid {
	return &Grid{Size: size, Overlap: overlap}
}
