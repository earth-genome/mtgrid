# mtgrid

[![Go CI](https://github.com/earth-genome/mtgrid/actions/workflows/smoke.yaml/badge.svg)](https://github.com/earth-genome/mtgrid/actions/workflows/smoke.yaml)

This is an implementation of the ESA [Major TOM](https://github.com/ESA-PhiLab/Major-TOM) equal area grid in Go. 


## Usage: 

Also see [tests](majortom/mtgrid_test.go) 

```go
package main

import (
    "github.com/earth-genome/mtgrid"
)

func main(){

	var someGeojson = ....
	
    fc, _ := geojson.UnmarshalFeatureCollection([]byte(someGeojson))
    
    g := fc.Features[0].Geometry
    p := g.(orb.Polygon)
    grid := NewGrid(320, true)
    cells, _ := grid.GenerateGridCells(&p)
	
	//do something with cells 
	for _, cell := range cells {
		
		printf("cell id %s", cell.Id())
    }
}


```

Before:
![Before](imgs/before.png)

After:
![After](./imgs/after.png)
