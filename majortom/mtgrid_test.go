package majortom

import (
	"fmt"
	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"testing"
)

//	var world = `{
//	 "type": "FeatureCollection",
//	 "features": [
//	   {
//	     "type": "Feature",
//	     "properties": {},
//	     "geometry": {
//	       "coordinates": [
//	         [
//	           [
//	             -180.0,
//	             -85.06
//	           ],
//	           [
//	             -180.0,
//	             85.06
//	           ],
//	           [
//	             180.0,
//	             85.06
//	           ],
//	           [
//	             180.0,
//	             -85.06
//	           ],
//	           [
//	             -180.0,
//	             -85.06
//	           ]
//	         ]
//	       ],
//	       "type": "Polygon"
//	     }
//	   }
//	 ]
//	}`
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
	fc2 := geojson.NewFeatureCollection()
	t.Logf("Cells: %v", len(cells))

	for _, f := range cells {
		nf := geojson.NewFeature(f)
		nf.Properties["id"] = geohash.EncodeWithPrecision(f.Bound().Center().Lat(), f.Bound().Center().Lon(), 7)
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
