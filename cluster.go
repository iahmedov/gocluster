package cluster

import (
	"math"

	"github.com/MadAppGang/kdbush"
)

// GeoCoordinates represent position in the Earth
type GeoCoordinates struct {
	Lon float64
	Lat float64
}

// all object, that you want to cluster should implement this protocol
type GeoPoint interface {
	GetCoordinates() GeoCoordinates
}

//Struct that implements clustered points
//could have only one point or set of points
type ClusterPoint struct {
	X, Y           float64
	visited        bool
	Id             int //Index for pint, Id for cluster
	NumPoints      int
	IncludedPoints []GeoPoint
}

func (cp *ClusterPoint) Coordinates() (float64, float64) {
	return cp.X, cp.Y
}

// Cluster struct get a list or stream of geo objects
// and produce all levels of clusters
// PointSize - pixel size of marker, affects clustering radius
// TileSize - size of tile in pixels, affects clustering radius
type Cluster struct {
	Epsilon      float64
	NodeSize     int
	ResultPoints []ClusterPoint

	ClusterIdxSeed int
	clusterIDLast  int
}

// Create new Cluster instance with default parameters:
// NodeSize is size of the KD-tree node, 64 by default. Higher means faster indexing but slower search, and vise versa.
func NewCluster(epsilon float64) *Cluster {
	return &Cluster{
		Epsilon:  epsilon,
		NodeSize: 64,
	}
}

// ClusterPoint get points and create multilevel clustered indexes
// All points should implement GeoPoint interface
// they are not copied, so you could not worry about memory efficiency
// And GetCoordinates called only once for each object, so you could calc it on the fly, if you need
func (c *Cluster) ClusterPoints(points []GeoPoint) error {

	//get digits number, start from next exponent
	//if we have 78, all cluster will start from 100...
	//if we have 986 points, all clusters ids will start from 1000
	c.ClusterIdxSeed = int(math.Pow(10, float64(digitsCount(len(points)))))
	c.clusterIDLast = c.ClusterIdxSeed

	clusters := translateGeoPointsToClusterPoints(points)
	tmpIndex := kdbush.NewBush(clustersToPoints(clusters), c.NodeSize)

	//create clusters for level up using just created index
	clusters = c.clusterize(clusters, tmpIndex)
	c.ResultPoints = make([]ClusterPoint, 0, len(clusters))
	for i := range clusters {
		cluster := *clusters[i]
		coordinates := ReverseMercatorProjection(cluster.X, cluster.Y)
		cluster.X = coordinates.Lon
		cluster.Y = coordinates.Lat
		c.ResultPoints = append(c.ResultPoints, cluster)
	}

	return nil
}

// AllClusters returns all cluster points
func (c *Cluster) AllClusters() []ClusterPoint {
	return c.ResultPoints
}

//clusterize points
func (c *Cluster) clusterize(points []*ClusterPoint, index *kdbush.KDBush) []*ClusterPoint {
	var result []*ClusterPoint
	r := c.Epsilon

	//iterate all clusters
	for pi := range points {
		p := points[pi]
		//skip points we have already clustered
		if p.visited {
			continue
		}
		// mark this point as visited
		p.visited = true

		//find all neighbours
		neighbourIds := index.Within(&kdbush.SimplePoint{X: p.X, Y: p.Y}, r)

		nPoints := p.NumPoints
		wx := p.X
		wy := p.Y

		var foundNeighbours []*ClusterPoint
		includedPoints := p.IncludedPoints

		for j := range neighbourIds {
			b := points[neighbourIds[j]]

			//Filter out neighbours, that are already processed (and processed point "p" as well)
			if !b.visited {
				wx += b.X
				wy += b.Y
				nPoints += b.NumPoints
				b.visited = true //set the zoom to skip in other iterations
				foundNeighbours = append(foundNeighbours, b)
				includedPoints = append(includedPoints, b.IncludedPoints...)
			}
		}
		newCluster := p

		//create new cluster
		if len(foundNeighbours) > 0 {
			newCluster = &ClusterPoint{}
			newCluster.X = wx / float64(nPoints)
			newCluster.Y = wy / float64(nPoints)
			newCluster.NumPoints = nPoints
			newCluster.visited = false
			newCluster.Id = c.clusterIDLast
			newCluster.IncludedPoints = includedPoints
			c.clusterIDLast += 1
		}
		result = append(result, newCluster)
	}
	return result
}

////////// End of Cluster implementation

//translate geopoints to ClusterPoints witrh projection coordinates
func translateGeoPointsToClusterPoints(points []GeoPoint) []*ClusterPoint {
	var result = make([]*ClusterPoint, len(points))
	for i, p := range points {
		cp := ClusterPoint{
			IncludedPoints: []GeoPoint{points[i]},
		}
		cp.visited = false
		cp.X, cp.Y = MercatorProjection(p.GetCoordinates())
		result[i] = &cp
		cp.NumPoints = 1
		cp.Id = i
	}
	return result
}

// longitude/latitude to spherical mercator in [0..1] range
func MercatorProjection(coordinates GeoCoordinates) (float64, float64) {
	x := coordinates.Lon/360.0 + 0.5
	sin := math.Sin(coordinates.Lat * math.Pi / 180.0)
	y := (0.5 - 0.25*math.Log((1+sin)/(1-sin))/math.Pi)
	if y < 0 {
		y = 0
	}
	if y > 1 {
		y = 1
	}
	return x, y
}
func ReverseMercatorProjection(x, y float64) GeoCoordinates {
	result := GeoCoordinates{}
	result.Lon = (x - 0.5) * 360
	y2 := (180 - y*360) * math.Pi / 180.0
	result.Lat = 360*math.Atan(math.Exp(y2))/math.Pi - 90
	return result
}

//count number of digits, for example 123356 will return 6
func digitsCount(a int) int {
	return int(math.Floor(math.Log10(math.Abs(float64(a))))) + 1
}

func clustersToPoints(points []*ClusterPoint) []kdbush.Point {
	result := make([]kdbush.Point, len(points))
	for i, v := range points {
		result[i] = v
	}
	return result
}
