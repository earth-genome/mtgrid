# API reference

The public API lives in the `majortom` package (`github.com/earth-genome/mtgrid/majortom`).

## Types

### `GridCell`

A grid cell represented as an `orb.Polygon` with a pre-computed geohash identifier.

- **`Id() string`** — Returns the cell's 11-character geohash ID, encoded from the polygon centroid at precision 11.

### `MajorTomGrid`

An equal-area geographical grid aligned with the [ESA Major TOM](https://github.com/ESA-PhiLab/Major-TOM) reference implementation. Latitude and longitude grid lines match the ESA `linspace` + `mod` scheme; the equator and prime meridian fall on grid lines where applicable.

## Functions

### `NewGrid(size uint64, overlap bool) *MajorTomGrid`

Construct a grid instance.

| Parameter | Description |
|-----------|-------------|
| `size` | Nominal cell edge length in **meters** (e.g. `320` for 320 m cells). |
| `overlap` | When `true`, generate additional cells offset by half the lat/lon spacing. When `false`, only primary (non-overlapping) cells are produced. |

## Methods

### `GenerateGridCells(geo orb.Geometry) ([]GridCell, error)`

Divide the area of interest into grid cells and return those that geometrically intersect the input geometry.

- Accepts any `orb.Geometry` value (e.g. `orb.Polygon`, `*orb.Polygon`, `orb.MultiPolygon`).
- Uses the geometry's bounding box to determine the row/column search space, then applies precise intersection checks via `github.com/tingold/orb-predicates`.
- Row processing runs concurrently; results are collected per row to minimize lock contention.
- Returns an error if the geometry is `nil`.

### `CellFromId(id string) (*GridCell, error)`

Retrieve a grid cell from its geohash ID.

- Computes the row/column index directly from the geohash centroid rather than scanning all cells in the area.
- If `id` is longer than 11 characters, it is **truncated to the first 11 characters** before lookup.
- Searches the computed cell and its immediate row/column neighbors to handle floating-point edge cases.
- When overlap mode is enabled, also checks overlap-offset cells.
- Returns an error if the geohash cannot be decoded or no matching cell is found.

### `MigrateCellId(oldId string) (*GridCell, error)`

Map a cell ID from a prior grid version to the current grid's **primary** (non-overlap) cell.

- Decodes the geohash to recover the approximate centroid of the old cell.
- Returns the primary cell in the current grid that contains that centroid.
- If `oldId` is longer than 11 characters, it is truncated to 11 characters. The ID must be at least 11 characters after truncation.
- Returns an error if the ID is too short or the geohash cannot be decoded.

## Cell IDs

All cell IDs are **11-character geohashes** (`geohashPrecision = 11`), derived from each cell polygon's centroid. This precision is used consistently for ID generation, lookup, and migration.
