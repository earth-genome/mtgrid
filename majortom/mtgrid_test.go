package majortom

import (
	"encoding/json"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"github.com/pierrre/assert"
	"github.com/pierrre/geohash"
	"os"
	"testing"
	"time"
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

	g := NewGrid(320, true)
	cell, err := g.CellFromId("dr19n8f7v6e")
	if err != nil {
		t.FailNow()
	} else {
		t.Logf("Expected: dr19n8f7v6e, Got: %s", cell.Id())
	}
	t.Log(string(wkt.Marshal(cell.Polygon)))
	box, err := geohash.Decode("dr19n8f7v6e")
	if err != nil {
		t.FailNow()
	}
	b := orb.Bound{
		Min: orb.Point{box.Lon.Min, box.Lat.Min},
		Max: orb.Point{box.Lon.Max, box.Lat.Max},
	}
	p := b.ToPolygon()
	t.Log(string(wkt.Marshal(p)))

	cells, _ := g.GenerateGridCells(&p)
	fc := geojson.NewFeatureCollection()
	for _, c := range cells {
		fc.Append(geojson.NewFeature(c.Polygon))
	}
	js, _ := fc.MarshalJSON()
	t.Log(string(js))
}

func TestSimple(t *testing.T) {

	fc, err := geojson.UnmarshalFeatureCollection([]byte(bigSouthampton))
	if err != nil {
		t.FailNow()
	}
	g := fc.Features[0].Geometry
	p := g.(orb.Polygon)
	grid := NewGrid(320, true)
	cells, err := grid.GenerateGridCells(&p)
	if err != nil {
		t.FailNow()
	}

	fc2 := geojson.NewFeatureCollection()
	t.Logf("Cells: %v", len(cells))

	lookup := make(map[string]bool)
	for _, f := range cells {
		_, exists := lookup[f.Id()]
		// fail on duplicates
		if exists {
			t.FailNow()
		}
		lookup[f.Id()] = true

		nf := geojson.NewFeature(f.Polygon)
		nf.Properties["id"] = f.Id()
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
	grid := NewGrid(320, true)
	smallerAoiCells, _ := grid.GenerateGridCells(&p)

	fc, err = geojson.UnmarshalFeatureCollection([]byte(bigSouthampton))
	if err != nil {
		t.FailNow()
	}
	g = fc.Features[0].Geometry
	p = g.(orb.Polygon)
	largerAoiCells, _ := grid.GenerateGridCells(&p)

	//t.Logf("largerAoi: %v", len(largerAoiCells))
	//t.Logf("smallerAoi: %v", len(smallerAoiCells))

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

	start := time.Now()
	fc, err := geojson.UnmarshalFeatureCollection([]byte(southampton))
	if err != nil {
		t.FailNow()
	}

	g := fc.Features[0].Geometry
	p := g.(orb.Polygon)

	grid := NewGrid(320, true)
	cells, _ := grid.GenerateGridCells(&p)
	t.Logf("Generated %v cells", len(cells))
	for _, cell := range cells {
		id := cell.Id()
		foundCell, _ := grid.CellFromId(id)
		assert.Equal(t, foundCell.Id(), cell.Id())
		assert.True(t, foundCell.Polygon.Equal(cell.Polygon))
	}
	end := time.Now()
	t.Logf("Completed in %v", end.Sub(start))

}

func TestTile(t *testing.T) {

	mtg := NewGrid(320, true)
	tile := maptile.Tile{
		X: uint32(5122),
		Y: uint32(8031),
		Z: maptile.Zoom(14),
	}
	p := tile.Bound().ToPolygon()
	cells, err := mtg.GenerateGridCells(&p)
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

// TestSmallGrid tests grids of different sizes to ensure that the IDs don't conflict
func TestSmallGrid(t *testing.T) {

	mtg := NewGrid(1, true)
	tile := maptile.Tile{
		X: uint32(5122),
		Y: uint32(8031),
		Z: maptile.Zoom(14),
	}
	p := tile.Bound().ToPolygon()
	cells, _ := mtg.GenerateGridCells(&p)

	ids := make(map[string]bool)
	for _, c := range cells {

		id := c.Id()
		_, contains := ids[id]
		if contains {
			t.Logf("Duplicate id found: %s", id)
			t.Fail()
		} else {
			ids[id] = true
		}
	}
}

func TestOddTile(t *testing.T) {
	g := NewGrid(320, true)
	cell, err := g.CellFromId("gcp0ywcrhk1")
	if err != nil {
		t.Fail()
	} else {
		t.Logf("Expected: gcp0ywcrhk1, Got: %s", cell.Id())
	}
	cell, err = g.CellFromId("gcp0ywcrhk1t24vzxu52")
	if err != nil {
		t.Fail()
	} else {
		t.Logf("Expected: gcp0ywcrhk1, Got: %s", cell.Id())
	}

}

func BenchmarkGenerateGridCells(b *testing.B) {
	fc, err := geojson.UnmarshalFeatureCollection([]byte(bigSouthampton))
	if err != nil {
		b.Fatal(err)
	}
	p := fc.Features[0].Geometry.(orb.Polygon)
	grid := NewGrid(320, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := grid.GenerateGridCells(&p)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateGridCellsNoOverlap(b *testing.B) {
	fc, err := geojson.UnmarshalFeatureCollection([]byte(bigSouthampton))
	if err != nil {
		b.Fatal(err)
	}
	p := fc.Features[0].Geometry.(orb.Polygon)
	grid := NewGrid(320, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := grid.GenerateGridCells(&p)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCellFromId(b *testing.B) {
	grid := NewGrid(320, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := grid.CellFromId("dr19n8f7v6e")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGridCellId(b *testing.B) {
	p := orb.Polygon{{
		{-76.34, 39.54},
		{-76.33, 39.54},
		{-76.33, 39.55},
		{-76.34, 39.55},
		{-76.34, 39.54},
	}}
	cell := newGridCell(p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cell.Id()
	}
}

func TestPythonCompatibility(t *testing.T) {
	poly := orb.Polygon{
		{
			{-76.33688798683391, 39.56892059705632},
			{-76.33688798683391, 39.54865173376891},
			{-76.30633471133426, 39.54865173376891},
			{-76.30633471133426, 39.56892059705632},
			{-76.33688798683391, 39.56892059705632},
		},
	}
	mg := NewGrid(320, true)
	cells, err := mg.GenerateGridCells(poly)
	if err != nil {
		t.Fatalf("failed to generate grid cells: %v", err)
	}

	// Load the output.geojson file
	file, err := os.Open("../python_output.geojson")
	if err != nil {
		t.Fatalf("failed to open output.geojson file: %v", err)
	}
	defer file.Close()

	// Decode the GeoJSON file
	featureCollection := geojson.NewFeatureCollection()
	if err := json.NewDecoder(file).Decode(&featureCollection); err != nil {
		t.Fatalf("failed to decode geojson: %v", err)
	}

	// Ensure the features match the grid cells
	assert.Equal(t, len(featureCollection.Features), len(cells))
	// Compare each feature geometry to the generated grid cells
	for _, feature := range featureCollection.Features {
		featurePolygon, ok := feature.Geometry.(orb.Polygon)
		if !ok {
			t.Errorf("geometry in feature is not a polygon: %v", feature.Geometry)
			continue
		}

		// Check if the feature exists in the grid cells
		found := false
		for _, cell := range cells {

			if featurePolygon.Equal(cell.Polygon) {
				assert.Equal(t, feature.Properties["cell_id"].(string), cell.Id())
				found = true
				break
			}
		}

		if !found {
			t.Errorf("feature geometry %v not found in grid cells", featurePolygon)
		}
	}

}
