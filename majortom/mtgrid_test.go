package majortom

import (
	"encoding/binary"
	"fmt"
	"github.com/apache/arrow/go/v16/arrow"
	"github.com/apache/arrow/go/v16/arrow/array"
	"github.com/apache/arrow/go/v16/arrow/memory"
	"github.com/apache/arrow/go/v16/parquet"
	"github.com/apache/arrow/go/v16/parquet/compress"
	"github.com/apache/arrow/go/v16/parquet/pqarrow"
	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/geojson"
	"io"
	"log"
	"os"
	"testing"
)

var world = `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {},
      "geometry": {
        "coordinates": [
          [
            [
              -180.0,
              -85.06
            ],
            [
              -180.0,
              85.06
            ],
            [
              180.0,
              85.06
            ],
            [
              180.0,
              -85.06
            ],
            [
              -180.0,
              -85.06
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
	fc, err := geojson.UnmarshalFeatureCollection([]byte(world))
	if err != nil {
		t.FailNow()
	}
	g := fc.Features[0].Geometry
	p := g.(orb.Polygon)
	grid := New(320, true)
	count := grid.CountCells(&p)
	fmt.Println(count)
	//49,923,554,414,992,355,441
	//5, 100, 000, 000, 000, 000

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

func TestWorld(t *testing.T) {

	f, _ := os.Open("world.geojson")
	data, _ := io.ReadAll(f)
	fc, _ := geojson.UnmarshalFeatureCollection(data)
	//fc, _ := geojson.UnmarshalFeatureCollection([]byte(southampton))

	//setup file output
	outFile, err := os.Create("majortom.parquet")
	if err != nil {
		log.Fatal("failed to open output file")
	}

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "geom", Type: arrow.BinaryTypes.Binary},
		{Name: "hash", Type: arrow.BinaryTypes.String},
	}, nil)
	props := parquet.NewWriterProperties(
		parquet.WithCompression(compress.Codecs.Snappy),
		parquet.WithRootName("spark_schema"),
		parquet.WithRootRepetition(parquet.Repetitions.Required),
	)
	pqWriter, err := pqarrow.NewFileWriter(schema, outFile, props, pqarrow.DefaultWriterProps())
	if err != nil {
		log.Fatal("failed to create output writer")
	}

	defer pqWriter.Close()

	pool := memory.NewGoAllocator()
	b := array.NewRecordBuilder(pool, schema)
	defer b.Release()
	//
	geochan := make(chan orb.Polygon)
	grid := New(320, true)
	p := fc.Features[0].Geometry.(orb.MultiPolygon)
	go grid.TilePolygonToChan(&p, geochan)
	count := 0
	for poly := range geochan {

		if poly == nil {
			continue
		}
		count++
		hash := geohash.EncodeWithPrecision(poly.Bound().Center().Lat(), poly.Bound().Center().Lon(), 7)
		blob, err := wkb.Marshal(poly, binary.BigEndian)
		if err != nil {
			t.FailNow()
		}
		err = writeRecord(b, pqWriter, blob, hash)
		if err != nil {
			t.FailNow()
		}
		if count%10000 == 0 {
			log.Printf("Processed %d polygons", count)
		}
	}
}

func writeRecord(b *array.RecordBuilder, pqWriter *pqarrow.FileWriter, geom []byte, hash string) error {

	b.Field(0).(*array.BinaryBuilder).Append(geom)
	b.Field(1).(*array.StringBuilder).AppendString(hash)

	rec := b.NewRecord()
	defer rec.Release()

	err := pqWriter.WriteBuffered(rec)
	if err != nil {
		return fmt.Errorf("failed to write to parquet file: %w", err)
	}

	return nil
}
