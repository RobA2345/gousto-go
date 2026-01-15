package main

type Nutrition struct {
	Calories string `json:"calories"`
	Protein  string `json:"protein"`
	Carbs    string `json:"carbs"`
	Fat      string `json:"fat"`
}

type Recipe struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Ingredients []string  `json:"ingredients"`
	Nutrition   Nutrition `json:"nutrition"`
	Tags        []string  `json:"tags"`
	ImageFront  string    `json:"image_front"`
	ImageBack   string    `json:"image_back"`
}

type Data struct {
	Recipes []Recipe `json:"recipes"`
}
