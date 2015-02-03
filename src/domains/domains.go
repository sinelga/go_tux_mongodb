package domains

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Blog struct {
	Title         string        `json:"title"`
	Author        string        `json:"author"`
	Id            bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Contents      string        `json:"contents"`
	Permanentlink string        `json:"permanentlink"`
	Imglink       string        `json:"imglink"`
	Extlink       string        `json:"extlink"`
	Pubdate       time.Time     `json:"pubdate"`
	Keywords      string        `json:"keywords"`
	Tags          string        `json:"tags"`
}
