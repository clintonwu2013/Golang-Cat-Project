package model

type CreateAdoptArticleRequest struct {
	City           string `json:"city"`
	Area           string `json:"area"`
	CatName        string `json:"catName"`
	CatAge         string `json:"catAge"`
	CatPersonality string `json:"catPersonality"`
	CatStory       string `json:"catStory"`
}

type AdoptPostModel struct {
	Id              int    `json:"id" db:"id"`
	Author_id       string `json:"authorId"`
	Title           string `json:"title"`
	City            string `json:"city"`
	Area            string `json:"area"`
	Cat_name        string `json:"catName"`
	Cat_age         string `json:"catAge"`
	Cat_personality string `json:"catPersonality"`
	Cat_story       string `json:"catStory"`
}
