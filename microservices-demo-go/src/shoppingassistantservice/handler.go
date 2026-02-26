package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/GoogleCloudPlatform/microservices-demo-go/src/shoppingassistantservice/internal/db"
	"github.com/GoogleCloudPlatform/microservices-demo-go/src/shoppingassistantservice/internal/llm"
)

type Handler struct {
	productStore db.ProductStore
	llmClient    llm.Client
	projectID    string
	location     string
}

type TalkRequest struct {
	Message string `json:"message"`
	Image   string `json:"image"` // URL
}

type TalkResponse struct {
	Content string `json:"content"`
}

func (h *Handler) talkToGemini(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TalkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Step 1: Get room description from Gemini Vision (or Mock)
	var images []string
	if req.Image != "" {
		images = []string{req.Image}
	}

	descriptionPrompt := "You are a professional interior designer, give me a detailed description of the style of the room in this image"
	descriptionResponse, err := h.llmClient.GenerateContent(ctx, descriptionPrompt, images)
	if err != nil {
		log.Printf("Error generating description: %v", err)
		http.Error(w, "Failed to generate description", http.StatusInternalServerError)
		return
	}

	// Step 2: Embed search prompt
	searchPrompt := fmt.Sprintf("This is the user's request: %s Find the most relevant items for that prompt, while matching style of the room described here: %s", req.Message, descriptionResponse)

	embeddingValues, err := h.llmClient.EmbedContent(ctx, searchPrompt)
	if err != nil {
		log.Printf("Error embedding content: %v", err)
		http.Error(w, "Failed to generate embedding", http.StatusInternalServerError)
		return
	}

	// Step 3: Vector Search using ProductStore
	products, err := h.productStore.SearchProducts(ctx, embeddingValues)
	if err != nil {
		log.Printf("Error searching products: %v", err)
		http.Error(w, "Failed to search products", http.StatusInternalServerError)
		return
	}

	var relevantDocs []string
	for _, p := range products {
		relevantDocs = append(relevantDocs, fmt.Sprintf("{id: %s, description: %s}", p.ID, p.Description))
	}

	// Step 4: Final Generation
	docsStr := strings.Join(relevantDocs, ", ")
	finalPrompt := fmt.Sprintf(`You are an interior designer that works for Online Boutique. You are tasked with providing recommendations to a customer on what they should add to a given room from our catalog. This is the description of the room: 
%s Here are a list of products that are relevant to it: %s Specifically, this is what the customer has asked for, see if you can accommodate it: %s Start by repeating a brief description of the room's design to the customer, then provide your recommendations. Do your best to pick the most relevant item out of the list of products provided, but if none of them seem relevant, then say that instead of inventing a new product. At the end of the response, add a list of the IDs of the relevant products in the following format for the top 3 results: [<first product ID>], [<second product ID>], [<third product ID>]`, descriptionResponse, docsStr, req.Message)

	finalContent, err := h.llmClient.GenerateContent(ctx, finalPrompt, nil)
	if err != nil {
		log.Printf("Error generating final response: %v", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TalkResponse{
		Content: finalContent,
	})
}
