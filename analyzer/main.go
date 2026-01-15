package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil) // Uses GOOGLE_API_KEY env var
	if err != nil {
		log.Fatal(err)
	}

	// Read existing data.json
	dataBytes, err := os.ReadFile("./data.json")
	if err != nil {
		log.Fatalf("Failed to read data.json: %v", err)
	}
	var data Data
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		log.Fatalf("Failed to unmarshal data.json: %v", err)
	}

	// Create a map of existing recipes by image_front to avoid duplicates
	existingRecipes := make(map[string]bool)
	for _, r := range data.Recipes {
		existingRecipes[r.ImageFront] = true
	}

	// List all files in the images directory
	files, err := os.ReadDir("./images")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()

		// Process only files that end with "_front.webp"
		if strings.HasSuffix(filename, "_front.webp") {
			imagePath := filepath.Join("./images", filename)

			// Skip if already processed
			if existingRecipes[imagePath] {
				// fmt.Printf("Skipping already processed: %s\n", filename)
				continue
			}

			fmt.Printf("Processing: %s\n", filename)

			// Read image data
			imgData, err := os.ReadFile(filepath.Join("./", imagePath))
			if err != nil {
				log.Printf("Failed to read image %s: %v", filename, err)
				continue
			}

			// Determine type and set prompt
			isHelloFresh := strings.HasPrefix(filename, "hf_")
			var prompt string
			var parts []*genai.Part

			if isHelloFresh {
				// For Hello Fresh, we need to read the back image as well for nutrition
				backFilename := strings.Replace(filename, "_front.webp", "_back.webp", 1)
				backImgData, err := os.ReadFile(filepath.Join("./images", backFilename))
				if err != nil {
					log.Printf("Failed to read back image %s for Hello Fresh nutrition: %v", backFilename, err)
					// Proceed with just front image, nutrition might be empty
					parts = []*genai.Part{
						genai.NewPartFromText(`Please extract the recipe data from this Hello Fresh card image into a JSON object.
						The JSON structure should match this Go struct:
						type Recipe struct {
							Title       string    json:"title"
							Ingredients []string  json:"ingredients" // List ingredients (measurements are likely not present)
							Nutrition   Nutrition json:"nutrition"
							Tags        []string  json:"tags"
						}
						type Nutrition struct {
							Calories string json:"calories"
							Protein  string json:"protein"
							Carbs    string json:"carbs"
							Fat      string json:"fat"
						}
						Extract all ingredients visible on the front of the card.
						For nutrition, look for values like Energy (kcal), Protein, Carbohydrate, Fat.
						Generate suitable tags based on the title and ingredients.
						`),
						genai.NewPartFromBytes(imgData, "image/webp"),
					}
				} else {
					// We have both images
					prompt = `Please extract the recipe data from these Hello Fresh card images (front and back) into a JSON object.
					The JSON structure should match this Go struct:
					type Recipe struct {
						Title       string    json:"title"
						Ingredients []string  json:"ingredients" // List ingredients from the front (measurements are likely not present)
						Nutrition   Nutrition json:"nutrition"
						Tags        []string  json:"tags"
					}
					type Nutrition struct {
						Calories string json:"calories"
						Protein  string json:"protein"
						Carbs    string json:"carbs"
						Fat      string json:"fat"
					}
					Extract all ingredients visible on the front of the card.
					Extract nutrition information from the back of the card (look for Energy (kcal), Protein, Carbohydrate, Fat).
					Generate suitable tags based on the title and ingredients.
					`
					parts = []*genai.Part{
						genai.NewPartFromText(prompt),
						genai.NewPartFromBytes(imgData, "image/webp"),
						genai.NewPartFromBytes(backImgData, "image/webp"),
					}
				}
			} else {
				// Gousto
				prompt = `Please extract the recipe data from this Gousto card image into a JSON object.
				The JSON structure should match this Go struct:
				type Recipe struct {
					Title       string    json:"title"
					Ingredients []string  json:"ingredients" // IMPORTANT: Include measurements with the ingredient name (e.g. "200g Chicken Breast", "1 Red Pepper")
					Nutrition   Nutrition json:"nutrition"
					Tags        []string  json:"tags"
				}
				type Nutrition struct {
					Calories string json:"calories"
					Protein  string json:"protein"
					Carbs    string json:"carbs"
					Fat      string json:"fat"
				}
				Extract all ingredients visible on the front of the card. Be sure to include the quantity/measurement for each ingredient if visible.
				For nutrition, look for values like Energy (kcal), Protein, Carbohydrate, Fat.
				Generate suitable tags based on the title and ingredients.
				`
				parts = []*genai.Part{
					genai.NewPartFromText(prompt),
					genai.NewPartFromBytes(imgData, "image/webp"),
				}
			}

			// Satisfy the []*genai.Content signature
			contents := []*genai.Content{
				genai.NewContentFromParts(parts, genai.RoleUser),
			}

			// Use a config for stricter JSON output
			config := &genai.GenerateContentConfig{
				ResponseMIMEType: "application/json",
			}

			resp, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, config)
			if err != nil {
				log.Printf("Failed to generate content for %s: %v", filename, err)
				continue
			}

			if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
				log.Printf("No content generated for %s", filename)
				continue
			}

			// Unmarshal into our Go struct
			var newRecipe Recipe
			responseText := resp.Candidates[0].Content.Parts[0].Text
			// Clean up potential markdown code blocks
			responseText = strings.TrimPrefix(responseText, "```json")
			responseText = strings.TrimPrefix(responseText, "```")
			responseText = strings.TrimSuffix(responseText, "```")
			responseText = strings.TrimSpace(responseText)

			// Handle potential array response
			if strings.HasPrefix(responseText, "[") {
				var recipes []Recipe
				if err := json.Unmarshal([]byte(responseText), &recipes); err != nil {
					log.Printf("JSON Unmarshal error (array) for %s: %v. Text: %s", filename, err, responseText)
					continue
				}
				if len(recipes) > 0 {
					newRecipe = recipes[0]
				} else {
					log.Printf("Empty recipe array returned for %s", filename)
					continue
				}
			} else {
				if err := json.Unmarshal([]byte(responseText), &newRecipe); err != nil {
					log.Printf("JSON Unmarshal error (object) for %s: %v. Text: %s", filename, err, responseText)
					continue
				}
			}

			// Fill in the missing fields
			newRecipe.ID = fmt.Sprintf("%03d", len(data.Recipes)+1)
			newRecipe.ImageFront = imagePath
			newRecipe.ImageBack = strings.Replace(imagePath, "_front.webp", "_back.webp", 1)

			// Append to data
			data.Recipes = append(data.Recipes, newRecipe)

			// Save after each successful process
			updatedData, _ := json.MarshalIndent(data, "", "  ")
			if err := os.WriteFile("./data.json", updatedData, 0644); err != nil {
				log.Printf("Failed to write data.json: %v", err)
			}

			fmt.Printf("Successfully added: %s (Type: %s)\n", newRecipe.Title, map[bool]string{true: "Hello Fresh", false: "Gousto"}[isHelloFresh])
		}
	}
}
