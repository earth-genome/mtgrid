package majortom

import (
	"fmt"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"github.com/pierrre/geohash"
	"testing"
)

//	var world = `{
//		 "type": "FeatureCollection",
//		 "features": [
//		   {
//		     "type": "Feature",
//		     "properties": {},
//		     "geometry": {
//		       "coordinates": [
//		         [
//		           [
//		             -180.0,
//		             -85.06
//		           ],
//		           [
//		             -180.0,
//		             85.06
//		           ],
//		           [
//		             180.0,
//		             85.06
//		           ],
//		           [
//		             180.0,
//		             -85.06
//		           ],
//		           [
//		             -180.0,
//		             -85.06
//		           ]
//		         ]
//		       ],
//		       "type": "Polygon"
//		     }
//		   }
//		 ]
//		}`
var bigSouthampton = `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {},
      "geometry": {
        "coordinates": [
          [
            [
              -76.35673421721803,
              39.55614384974018
            ],
            [
              -76.35673421721803,
              39.53123810591927
            ],
            [
              -76.3131967920373,
              39.53123810591927
            ],
            [
              -76.3131967920373,
              39.55614384974018
            ],
            [
              -76.35673421721803,
              39.55614384974018
            ]
          ]
        ],
        "type": "Polygon"
      }
    }
  ]
}`

var southampton = `
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {},
      "geometry": {
        "coordinates": [
          [
            [
              -76.34311254543559,
              39.548984543610345
            ],
            [
              -76.34311254543559,
              39.53895755383081
            ],
            [
              -76.3260548926393,
              39.53895755383081
            ],
            [
              -76.3260548926393,
              39.548984543610345
            ],
            [
              -76.34311254543559,
              39.548984543610345
            ]
          ]
        ],
        "type": "Polygon"
      }
    }
  ]
}
`

func TestGridCell_Id(t *testing.T) {

	g := New(320, true)
	cell, err := g.CellFromId("gcrtrujj09r3gfd5jzz5")
	if err != nil {
		t.FailNow()
	} else {
		t.Logf("Expected: gcrtrujj09r3gfd5jzz5, Got: %s", cell.Id())
	}
	t.Log(string(wkt.Marshal(cell.Polygon)))
	box, err := geohash.Decode("gcrtrujj09r3gfd5jzz5")
	if err != nil {
		t.FailNow()
	}
	b := orb.Bound{
		Min: orb.Point{box.Lon.Min, box.Lat.Min},
		Max: orb.Point{box.Lon.Max, box.Lat.Max},
	}
	p := b.ToPolygon()
	t.Log(string(wkt.Marshal(p)))

	cells, _ := g.TilePolygon(&p)
	fc := geojson.NewFeatureCollection()
	for _, c := range cells {
		fc.Append(geojson.NewFeature(c.Polygon))
	}
	json, _ := fc.MarshalJSON()
	t.Log(string(json))
}

func TestCount(t *testing.T) {
	fc, err := geojson.UnmarshalFeatureCollection([]byte(bigSouthampton))
	if err != nil {
		t.FailNow()
	}
	g := fc.Features[0].Geometry
	p := g.(orb.Polygon)
	grid := New(320, true)
	count := grid.CountCells(&p)
	fmt.Println(count)
}

func TestSimple(t *testing.T) {

	fc, err := geojson.UnmarshalFeatureCollection([]byte(bigSouthampton))
	if err != nil {
		t.FailNow()
	}
	g := fc.Features[0].Geometry
	p := g.(orb.Polygon)
	grid := New(320, true)
	cells, err := grid.TilePolygon(&p)
	if err != nil {
		t.FailNow()
	}
	fc2 := geojson.NewFeatureCollection()
	t.Logf("Cells: %v", len(cells))

	for _, f := range cells {
		nf := geojson.NewFeature(f)
		nf.Properties["id"] = geohash.Encode(f.Bound().Center().Lat(), f.Bound().Center().Lon(), 20)
		fc2.Append(nf)
	}
	js, _ := fc2.MarshalJSON()
	t.Log(string(js))
}

// tests that 2 different polygons will result in aligned grid cells
func TestOffsets(t *testing.T) {

	fc, err := geojson.UnmarshalFeatureCollection([]byte(southampton))
	if err != nil {
		t.FailNow()
	}
	g := fc.Features[0].Geometry
	p := g.(orb.Polygon)
	grid := New(320, true)
	smallerAoiCells, _ := grid.TilePolygon(&p)

	fc, err = geojson.UnmarshalFeatureCollection([]byte(bigSouthampton))
	if err != nil {
		t.FailNow()
	}
	g = fc.Features[0].Geometry
	p = g.(orb.Polygon)
	largerAoiCells, _ := grid.TilePolygon(&p)

	t.Logf("largerAoi: %v", len(largerAoiCells))
	t.Logf("smallerAoi: %v", len(smallerAoiCells))
	//assert that all cells in the small aoi are also in the big aoi
	for _, cell := range smallerAoiCells {
		found := false
		for _, cell2 := range largerAoiCells {

			if cell2.Polygon.Equal(cell.Polygon) {
				found = true
			}
		}
		if !found {
			t.Log("cell was not found")
			t.Fail()
		}

	}

}

func TestIds(t *testing.T) {

	fc, err := geojson.UnmarshalFeatureCollection([]byte(southampton))
	if err != nil {
		t.FailNow()
	}
	g := fc.Features[0].Geometry
	p := g.(orb.Polygon)
	grid := New(320, true)
	cells, err := grid.TilePolygon(&p)
	if err != nil {
		t.FailNow()
	}

	for _, cell := range cells {
		id := cell.Id()
		foundCell, err := grid.CellFromId(id)
		if err != nil {
			t.Logf("error getting cell: %v", err)
			t.FailNow()
		}
		if foundCell == nil {
			t.Logf("error getting cell: %v", err)
			t.FailNow()
		}
		if !foundCell.Polygon.Equal(cell.Polygon) {
			t.Logf("cells are not equal!")
			t.FailNow()
		}
	}

}

func TestTile(t *testing.T) {

	mtg := New(320, true)
	tile := maptile.Tile{
		X: uint32(5122),
		Y: uint32(8031),
		Z: maptile.Zoom(14),
	}
	p := tile.Bound().ToPolygon()
	cells, err := mtg.TilePolygon(&p)
	if err != nil {
		t.FailNow()
	}
	gridFc := geojson.NewFeatureCollection()
	for _, cell := range cells {
		feat := geojson.NewFeature(cell.Polygon)
		//should be length 20
		feat.ID = cell.Id()
		feat.Properties["lon"] = cell.Bound().Center().Lon()
		feat.Properties["lat"] = cell.Bound().Center().Lat()
		feat.Properties["id"] = feat.ID
		gridFc.Append(feat)
	}
	js, _ := gridFc.MarshalJSON()
	print(string(js))

}
