package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	cluster "github.com/iahmedov/gocluster"
)

type TestPoint struct {
	Type       string
	Properties struct {
		//we don't need other data
		Name string
	}
	Geometry struct {
		Coordinates []float64
	}
}

func (tp *TestPoint) GetCoordinates() cluster.GeoCoordinates {
	return cluster.GeoCoordinates{
		Lon: tp.Geometry.Coordinates[0],
		Lat: tp.Geometry.Coordinates[1],
	}
}

//type MercatorPoint struct {
//	Cluster cluster.ClusterPoint
//	MercatorX int
//	MercatorY int
//}
//
//func mercator(p cluster.ClusterPoint) MercatorPoint {
//	mp := MercatorPoint{}
//	mp.Cluster = p
//	mp.MercatorX =
//
//}

func importData(filename string) []*TestPoint {
	var points = struct {
		Type     string
		Features []*TestPoint
	}{}
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	json.Unmarshal(raw, &points)
	return points.Features
}

type simplePoint struct {
	Lon, Lat float64
}

func (sp simplePoint) GetCoordinates() cluster.GeoCoordinates {
	return cluster.GeoCoordinates{sp.Lon, sp.Lat}
}

func main() {
	points := importData("./testdata/places.json")

	geoPoints := make([]cluster.GeoPoint, len(points))
	for i := range points {
		geoPoints[i] = points[i]
	}

	// Zoom range is limited by 0 to 21
	// PointSize - pixel size of marker, affects clustering radius
	// TileSize - size of tile in pixels, affects clustering radius
	zoom := 4
	pointSize := 60
	tileSize := 256
	z := 1 << uint64(zoom)
	epsilon := float64(pointSize) / float64(tileSize*z)

	c := cluster.NewCluster(epsilon)
	c.ClusterPoints(geoPoints)
	result := c.AllClusters()

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(resultJSON))
}
