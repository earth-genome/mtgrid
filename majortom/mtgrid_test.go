package majortom

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"github.com/pierrre/assert"
	"github.com/pierrre/geohash"
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
	cell, err := g.CellFromId("dr19n8ggbf9")
	if err != nil {
		t.FailNow()
	} else {
		t.Logf("Expected: dr19n8ggbf9, Got: %s", cell.Id())
	}
	t.Log(string(wkt.Marshal(cell.Polygon)))
	box, err := geohash.Decode("dr19n8ggbf9")
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

			if cell2.Equal(cell.Polygon) {
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
		assert.True(t, foundCell.Equal(cell.Polygon))
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
	cell, err := g.CellFromId("gcp0yqzxpk4")
	if err != nil {
		t.Fail()
	} else {
		t.Logf("Expected: gcp0yqzxpk4, Got: %s", cell.Id())
	}
	cell, err = g.CellFromId("gcp0yqzxpk4t24vzxu52")
	if err != nil {
		t.Fail()
	} else {
		t.Logf("Expected: gcp0yqzxpk4, Got: %s", cell.Id())
	}
	_ = cell
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

// esaLatitudes returns the reference latitude grid lines using the ESA linspace+mod approach.
func esaLatitudes(distKm float64) []float64 {
	const earthRadiusKm = 6378.137
	numDivisions := int(math.Ceil(math.Pi * earthRadiusKm / distKm))
	lats := make([]float64, numDivisions)
	step := 180.0 / float64(numDivisions)
	for i := 0; i < numDivisions; i++ {
		v := -90.0 + float64(i)*step
		v = math.Mod(v, 180)
		if v < 0 {
			v += 180
		}
		lats[i] = v - 90
	}
	sort.Float64s(lats)
	return lats
}

// esaLongitudes returns the reference longitude grid lines for a given latitude
// using the ESA linspace+mod approach.
func esaLongitudes(lat, distKm float64) []float64 {
	const earthRadiusKm = 6378.137
	circumference := 2 * math.Pi * earthRadiusKm * math.Cos(lat*math.Pi/180)
	numDivisions := int(math.Ceil(circumference / distKm))
	lons := make([]float64, numDivisions)
	step := 360.0 / float64(numDivisions)
	for i := 0; i < numDivisions; i++ {
		v := -180.0 + float64(i)*step
		v = math.Mod(v, 360)
		if v < 0 {
			v += 360
		}
		lons[i] = v - 180
	}
	sort.Float64s(lons)
	return lons
}

func egLatitudes(grid *MajorTomGrid) []float64 {
	lats := make([]float64, grid.rowCount)
	for i := int64(0); i < grid.rowCount; i++ {
		lats[i] = grid.rowLat(i)
	}
	return lats
}

func egLongitudes(grid *MajorTomGrid, lat float64) []float64 {
	ls := grid.lonSpacing(lat)
	lo := grid.lonOffset(ls)
	latRad := toRad(min(max(lat, -89), 89))
	nCols := int(math.Ceil(2 * math.Pi * radius * math.Cos(latRad) / float64(grid.Size)))
	lons := make([]float64, nCols)
	for i := 0; i < nCols; i++ {
		lons[i] = grid.colLon(i, ls, lo)
	}
	return lons
}

func TestESALatitudeAlignment(t *testing.T) {
	for _, distKm := range []float64{5, 10, 50, 100} {
		distM := uint64(distKm * 1000)
		grid := NewGrid(distM, false)
		esaLats := esaLatitudes(distKm)
		egLats := egLatitudes(grid)

		if len(esaLats) != len(egLats) {
			t.Fatalf("dist=%vkm: latitude count mismatch (%d vs %d)", distKm, len(esaLats), len(egLats))
		}
		for i := range esaLats {
			if math.Abs(egLats[i]-esaLats[i]) > 1e-10 {
				t.Fatalf("dist=%vkm: latitude[%d] differs: eg=%v esa=%v", distKm, i, egLats[i], esaLats[i])
			}
		}
	}
}

func TestESALongitudeAlignment(t *testing.T) {
	for _, distKm := range []float64{5, 10, 50, 100} {
		distM := uint64(distKm * 1000)
		grid := NewGrid(distM, false)
		for _, testLat := range []float64{0.0, 30.0, 45.0, 60.0} {
			esaLons := esaLongitudes(testLat, distKm)
			egLons := egLongitudes(grid, testLat)

			if len(esaLons) != len(egLons) {
				t.Fatalf("dist=%vkm, lat=%v: longitude count mismatch (%d vs %d)", distKm, testLat, len(esaLons), len(egLons))
			}
			for i := range esaLons {
				if math.Abs(egLons[i]-esaLons[i]) > 1e-10 {
					t.Fatalf("dist=%vkm, lat=%v: longitude[%d] differs: eg=%v esa=%v", distKm, testLat, i, egLons[i], esaLons[i])
				}
			}
		}
	}
}

func TestEquatorOnGridLine(t *testing.T) {
	for _, distKm := range []float64{5, 7, 10, 13, 50, 100} {
		distM := uint64(distKm * 1000)
		grid := NewGrid(distM, false)
		lats := egLatitudes(grid)
		found := false
		for _, lat := range lats {
			if lat == 0.0 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("dist=%vkm: equator (0.0) should be a grid line", distKm)
		}
	}
}

func TestPrimeMeridianOnGridLine(t *testing.T) {
	for _, distKm := range []float64{5, 7, 10, 13, 50, 100} {
		distM := uint64(distKm * 1000)
		grid := NewGrid(distM, false)
		for _, testLat := range []float64{0.0, 30.0, 60.0} {
			lons := egLongitudes(grid, testLat)
			found := false
			for _, lon := range lons {
				if lon == 0.0 {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("dist=%vkm, lat=%v: prime meridian (0.0) should be a grid line", distKm, testLat)
			}
		}
	}
}

func TestMigrateCellIdContainsOldCentroid(t *testing.T) {
	grid := NewGrid(320, true)
	oldIds := []string{"dr18zj1ntew", "dr19n8zgg4e", "s000003037z", "6r32gxpn0w4"}
	for _, oldId := range oldIds {
		newCell, err := grid.MigrateCellId(oldId)
		if err != nil {
			t.Fatalf("MigrateCellId(%s) error: %v", oldId, err)
		}
		box, err := geohash.Decode(oldId)
		if err != nil {
			t.Fatalf("geohash.Decode(%s) error: %v", oldId, err)
		}
		lat := box.Lat.Mid()
		lon := box.Lon.Mid()
		b := newCell.Polygon.Bound()
		if lon < b.Min.Lon() || lon > b.Max.Lon() || lat < b.Min.Lat() || lat > b.Max.Lat() {
			t.Fatalf("New cell for old ID %s does not contain decoded centroid (%.6f, %.6f)", oldId, lat, lon)
		}
	}
}

func TestMigrateCellIdIsPrimary(t *testing.T) {
	grid := NewGrid(320, true)
	newCell, err := grid.MigrateCellId("dr18zj1ntew")
	if err != nil {
		t.Fatalf("MigrateCellId error: %v", err)
	}
	if len(newCell.Id()) != 11 {
		t.Fatalf("Expected cell ID of length 11, got %d", len(newCell.Id()))
	}
}

func TestMigrateCellIdInvalidShort(t *testing.T) {
	grid := NewGrid(320, true)
	_, err := grid.MigrateCellId("short")
	if err == nil {
		t.Fatal("Expected error for short cell ID")
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
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close file: %v", err)
		}
	}()

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

type crossLangCell struct {
	Id     string      `json:"id"`
	Coords [][]float64 `json:"coords"`
}

type crossLangCase struct {
	Count   int             `json:"count"`
	Cells   []crossLangCell `json:"cells"`
	Config  struct {
		D       uint64 `json:"d"`
		Overlap bool   `json:"overlap"`
	} `json:"config"`
	Polygon [][]float64 `json:"polygon"`
}

func TestCrossLanguageCompatibility(t *testing.T) {
	data, err := os.ReadFile("../testdata/cross_language_reference.json")
	if err != nil {
		t.Fatalf("failed to read cross-language reference: %v", err)
	}

	var cases map[string]crossLangCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed to parse reference JSON: %v", err)
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ring := make(orb.Ring, len(tc.Polygon))
			for i, pt := range tc.Polygon {
				ring[i] = orb.Point{pt[0], pt[1]}
			}
			poly := orb.Polygon{ring}

			grid := NewGrid(tc.Config.D, tc.Config.Overlap)
			cells, err := grid.GenerateGridCells(poly)
			if err != nil {
				t.Fatalf("GenerateGridCells error: %v", err)
			}

			if len(cells) != tc.Count {
				t.Fatalf("cell count mismatch: Go=%d Python=%d", len(cells), tc.Count)
			}

			goIds := make(map[string]GridCell, len(cells))
			for _, c := range cells {
				goIds[c.Id()] = c
			}

			for _, pyCell := range tc.Cells {
				goCell, ok := goIds[pyCell.Id]
				if !ok {
					t.Errorf("Python cell ID %s not found in Go output", pyCell.Id)
					continue
				}

				goBound := goCell.Polygon.Bound()
				pyMinLon := pyCell.Coords[0][0]
				pyMinLat := pyCell.Coords[0][1]
				pyMaxLon := pyCell.Coords[2][0]
				pyMaxLat := pyCell.Coords[2][1]

				if math.Abs(goBound.Min.Lon()-pyMinLon) > 1e-8 ||
					math.Abs(goBound.Min.Lat()-pyMinLat) > 1e-8 ||
					math.Abs(goBound.Max.Lon()-pyMaxLon) > 1e-8 ||
					math.Abs(goBound.Max.Lat()-pyMaxLat) > 1e-8 {
					t.Errorf("Cell %s coordinates differ:\n  Go:     [%.12f,%.12f]-[%.12f,%.12f]\n  Python: [%.12f,%.12f]-[%.12f,%.12f]",
						pyCell.Id,
						goBound.Min.Lon(), goBound.Min.Lat(), goBound.Max.Lon(), goBound.Max.Lat(),
						pyMinLon, pyMinLat, pyMaxLon, pyMaxLat)
				}
			}
		})
	}
}
