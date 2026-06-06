package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/pdcgo/san_collection/san_config"
	"github.com/pdcgo/shared/db_models"
	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

const (
	location  = "asia-southeast2"
	model     = "multimodalembedding@001"
	dimension = 1408
)

type ReindexConfig struct {
	Dsn       string `env:"REINDEX_DSN"` // https://pdcgudang.et.r.appspot.com
	ProjectID string `env:"GOOGLE_CLOUD_PROJECT"`
}

type ReindexFunc cli.ActionFunc

func NewReindexFunc(
	db *gorm.DB,
) ReindexFunc {
	return func(ctx context.Context, c *cli.Command) error {
		cfg := ReindexConfig{}

		if err := san_config.LoadFromEnv(&cfg); err != nil {
			return err
		}

		// creating client
		client, err := aiplatform.NewPredictionClient(ctx)
		if err != nil {
			log.Fatalf("create client: %v", err)
		}
		defer client.Close()

		modelEndpoint := fmt.Sprintf(
			"projects/%s/locations/%s/publishers/google/models/%s",
			cfg.ProjectID, location, model,
		)

		rows, err := db.
			WithContext(ctx).
			Model(&db_models.Product{}).
			Limit(10).
			Rows()

		if err != nil {
			return err
		}

		defer rows.Close()

		for rows.Next() {
			var product db_models.Product
			if err := db.ScanRows(rows, &product); err != nil {
				return err
			}

			if len(product.Image) == 0 {
				slog.Warn("product has no image", "product", product)
				continue
			}

			imageUri := fmt.Sprintf("%s/v1/assets/get?id=%s", cfg.Dsn, product.Image[0])

			res, err := http.Get(imageUri)
			if err != nil {
				return err
			}

			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("product %s image %s not found", product.RefID, product.Image[0])
			}

			imageBytes, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}

			res.Body.Close()

			// embedding image
			imageEmb, err := embedImage(ctx, client, modelEndpoint, imageBytes)
			if err != nil {
				log.Fatalf("embed image: %v", err)
			}
			fmt.Printf("Image embedding: %d dims, first 5: %v\n",
				len(imageEmb), imageEmb[:5])

			// embeding text
			textEmb, err := embedText(ctx, client, modelEndpoint, product.Name)
			if err != nil {
				log.Fatalf("embed text: %v", err)
			}
			fmt.Printf("Text embedding:  %d dims, first 5: %v\n",
				len(textEmb), textEmb[:5])

			slog.Info("generating vector", "image", imageUri)
		}

		return nil
	}
}

func embedImage(
	ctx context.Context,
	client *aiplatform.PredictionClient,
	endpoint string,
	imageBytes []byte,
) ([]float32, error) {
	// Encode ke base64
	imageB64 := base64.StdEncoding.EncodeToString(imageBytes)

	// Build instance: {"image": {"bytesBase64Encoded": "..."}}
	instance, err := structpb.NewStruct(map[string]interface{}{
		"image": map[string]interface{}{
			"bytesBase64Encoded": imageB64,
		},
	})
	if err != nil {
		return nil, err
	}

	// Parameters: {"dimension": 1408}
	params, err := structpb.NewStruct(map[string]interface{}{
		"dimension": dimension,
	})
	if err != nil {
		return nil, err
	}

	// Call Predict API
	resp, err := client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint:   endpoint,
		Instances:  []*structpb.Value{structpb.NewStructValue(instance)},
		Parameters: structpb.NewStructValue(params),
	})
	if err != nil {
		return nil, err
	}

	return extractEmbedding(resp, "imageEmbedding")
}

// embedText generate embedding dari teks
func embedText(
	ctx context.Context,
	client *aiplatform.PredictionClient,
	endpoint string,
	text string,
) ([]float32, error) {
	// Build instance: {"text": "..."}
	instance, err := structpb.NewStruct(map[string]interface{}{
		"text": text,
	})
	if err != nil {
		return nil, err
	}

	params, err := structpb.NewStruct(map[string]interface{}{
		"dimension": dimension,
	})
	if err != nil {
		return nil, err
	}

	resp, err := client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint:   endpoint,
		Instances:  []*structpb.Value{structpb.NewStructValue(instance)},
		Parameters: structpb.NewStructValue(params),
	})
	if err != nil {
		return nil, err
	}

	return extractEmbedding(resp, "textEmbedding")
}

// extractEmbedding parse []float32 dari response API
func extractEmbedding(resp *aiplatformpb.PredictResponse, fieldName string) ([]float32, error) {
	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("no predictions")
	}

	pred := resp.Predictions[0].GetStructValue()
	field, ok := pred.Fields[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found", fieldName)
	}

	list := field.GetListValue()
	result := make([]float32, len(list.Values))
	for i, v := range list.Values {
		result[i] = float32(v.GetNumberValue())
	}
	return result, nil
}
