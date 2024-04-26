package mtgrid

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"testing"
)

var southampton = `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {},
      "geometry": {
        "coordinates": [
          [
            [
              -76.33381255765352,
              39.54768282695619
            ],
            [
              -76.33528125750044,
              39.54425881290615
            ],
            [
              -76.33241265450579,
              39.5431880932
            ],
            [
              -76.33064558524046,
              39.54674515650569
            ],
            [
              -76.33381255765352,
              39.54768282695619
            ]
          ]
        ],
        "type": "Polygon"
      }
    }
  ]
}`

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
	for _, f := range cells {
		fc2.Append(geojson.NewFeature(f))
	}
	js, _ := fc2.MarshalJSON()
	print(string(js))

}
