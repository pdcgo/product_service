package search

import (
	"context"
	"encoding/base64"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// EmbeddingClient untuk generate embedding pakai multimodalembedding@001
type EmbeddingClient struct {
	client    *aiplatform.PredictionClient
	endpoint  string
	dimension int
}

// NewEmbeddingClient bikin client untuk Vertex AI embedding
func NewEmbeddingClient(ctx context.Context, projectID, location string, dimension int) (*EmbeddingClient, error) {
	apiEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
	client, err := aiplatform.NewPredictionClient(ctx,
		option.WithEndpoint(apiEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("create prediction client: %w", err)
	}

	modelEndpoint := fmt.Sprintf(
		"projects/%s/locations/%s/publishers/google/models/multimodalembedding@001",
		projectID, location,
	)

	return &EmbeddingClient{
		client:    client,
		endpoint:  modelEndpoint,
		dimension: dimension,
	}, nil
}

// Close menutup koneksi
func (e *EmbeddingClient) Close() error {
	return e.client.Close()
}

// EmbedImage generate embedding dari bytes gambar
func (e *EmbeddingClient) EmbedImage(ctx context.Context, imageBytes []byte) ([]float32, error) {
	imageB64 := base64.StdEncoding.EncodeToString(imageBytes)

	instance, err := structpb.NewStruct(map[string]interface{}{
		"image": map[string]interface{}{
			"bytesBase64Encoded": imageB64,
		},
	})
	if err != nil {
		return nil, err
	}

	params, _ := structpb.NewStruct(map[string]interface{}{
		"dimension": e.dimension,
	})

	resp, err := e.client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint:   e.endpoint,
		Instances:  []*structpb.Value{structpb.NewStructValue(instance)},
		Parameters: structpb.NewStructValue(params),
	})
	if err != nil {
		return nil, fmt.Errorf("predict: %w", err)
	}

	return extractEmbedding(resp, "imageEmbedding")
}

// EmbedText generate embedding dari teks
func (e *EmbeddingClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	instance, err := structpb.NewStruct(map[string]interface{}{
		"text": text,
	})
	if err != nil {
		return nil, err
	}

	params, _ := structpb.NewStruct(map[string]interface{}{
		"dimension": e.dimension,
	})

	resp, err := e.client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint:   e.endpoint,
		Instances:  []*structpb.Value{structpb.NewStructValue(instance)},
		Parameters: structpb.NewStructValue(params),
	})
	if err != nil {
		return nil, fmt.Errorf("predict: %w", err)
	}

	return extractEmbedding(resp, "textEmbedding")
}

func extractEmbedding(resp *aiplatformpb.PredictResponse, field string) ([]float32, error) {
	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("no predictions")
	}
	pred := resp.Predictions[0].GetStructValue()
	listVal := pred.Fields[field].GetListValue()
	result := make([]float32, len(listVal.Values))
	for i, v := range listVal.Values {
		result[i] = float32(v.GetNumberValue())
	}
	return result, nil
}
