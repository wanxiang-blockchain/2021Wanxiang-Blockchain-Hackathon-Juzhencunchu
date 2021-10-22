package xml

import (
	"encoding/xml"
	"time"
)

type ListObjectResultContentOwner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}
type ListObjectResultContent struct {
	XMLName      xml.Name                      `xml:"Contents"`
	Key          string                        `xml:"Key"`
	LastModified time.Time                     `xml:"LastModified"`
	ETag         string                        `xml:"ETag"`
	Size         int                           `xml:"Size"`
	StorageClass string                        `xml:"StorageClass"`
	Owner        *ListObjectResultContentOwner `xml:"Owner"`
}

type ListObjectResult struct {
	XMLName     xml.Name                   `xml:"ListBucketResult"`
	Xmlns       string                     `xml:"xmlns,attr"`
	Name        string                     `xml:"Name"`
	Prefix      string                     `xml:"Prefix"`
	Marker      string                     `xml:"Marker"`
	MaxKeys     int                        `xml:"MaxKeys"`
	IsTruncated bool                       `xml:"IsTruncated"`
	Contents    []*ListObjectResultContent `xml:"Contents"`
}
