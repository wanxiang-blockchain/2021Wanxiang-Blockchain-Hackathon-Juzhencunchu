package xml

import (
	"encoding/xml"
	"time"
)

type CreateBucket struct {
	XMLName  xml.Name `xml:"CreateBucketConfiguration"`
	Location string   `xml:"LocationConstraint"`
}
type Bucket struct {
	XMLName      xml.Name  `xml:"Bucket"`
	CreationDate time.Time `xml:"CreationDate"`
	Name         string    `xml:"Name"`
}
type Buckets struct {
	XMLName xml.Name `xml:"Buckets"`
	Bucket  []Bucket `xml:"Bucket"`
}
type Owner struct {
	XMLName     xml.Name `xml:"Owner"`
	DisplayName string   `xml:"DisplayName"`
	Id          string   `xml:"ID"`
}
type ListBucket struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Buckets Buckets  `xml:"Buckets"`
	Owner   Owner    `xml:"Owner"`
}
