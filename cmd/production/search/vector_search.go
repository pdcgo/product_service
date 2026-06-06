package search

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
)

// ============================================================
// PENJELASAN ARSITEKTUR VECTOR SEARCH
// ============================================================
//
// Vertex AI Vector Search punya 2 komponen utama:
//
//   1. INDEX - tempat penyimpanan vector
//      - Cuma "wadah", belum bisa di-query
//      - Bisa di-update lewat Upsert/Remove datapoints
//      - Dipakai oleh IndexClient
//
//   2. INDEX ENDPOINT - tempat deploy index supaya bisa di-query
//      - Mirip server endpoint
//      - 1 endpoint bisa deploy banyak index (DeployedIndex)
//      - Dipakai oleh MatchClient
//
// Jadi alurnya:
//   - Save vector  → ke INDEX (pakai IndexClient.UpsertDatapoints)
//   - Search       → ke INDEX ENDPOINT (pakai MatchClient.FindNeighbors)
// ============================================================

// VectorSearchClient wrapper untuk operasi save & search
type VectorSearchClient struct {
	indexClient *aiplatform.IndexClient // untuk save/upsert
	matchClient *aiplatform.MatchClient // untuk search

	// Resource names yang sudah dibuat di Cloud Console
	indexName         string // format: projects/.../indexes/123456
	indexEndpointName string // format: projects/.../indexEndpoints/789012
	deployedIndexID   string // ID custom yang dipilih saat deploy
}

// VectorSearchConfig konfigurasi untuk client
type VectorSearchConfig struct {
	Location          string // misal "asia-southeast1"
	IndexName         string // full resource name dari index
	IndexEndpointName string // full resource name dari endpoint
	DeployedIndexID   string // ID deployment
}

// NewVectorSearchClient bikin client baru
func NewVectorSearchClient(ctx context.Context, cfg VectorSearchConfig) (*VectorSearchClient, error) {
	// apiEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", cfg.Location)

	// IndexClient untuk operasi MANAGE index (upsert, remove datapoints)
	indexClient, err := aiplatform.NewIndexClient(ctx) // option.WithEndpoint(apiEndpoint),

	if err != nil {
		return nil, fmt.Errorf("create index client: %w", err)
	}

	// MatchClient untuk operasi QUERY (find nearest neighbors)
	matchClient, err := aiplatform.NewMatchClient(ctx) // option.WithEndpoint(apiEndpoint),

	if err != nil {
		indexClient.Close()
		return nil, fmt.Errorf("create match client: %w", err)
	}

	return &VectorSearchClient{
		indexClient:       indexClient,
		matchClient:       matchClient,
		indexName:         cfg.IndexName,
		indexEndpointName: cfg.IndexEndpointName,
		deployedIndexID:   cfg.DeployedIndexID,
	}, nil
}

// Close menutup semua koneksi
func (v *VectorSearchClient) Close() error {
	v.matchClient.Close()
	return v.indexClient.Close()
}

// ============================================================
// SAVE OPERATIONS (Upsert)
// ============================================================

// Datapoint = 1 record dalam vector search
// ID unik + vector + metadata opsional (untuk filter)
type Datapoint struct {
	ID        string              // unique identifier (misal product ID)
	Vector    []float32           // vector hasil embedding
	Restricts []DatapointRestrict // metadata untuk filter saat search
}

// DatapointRestrict metadata yang bisa dipakai untuk filter
// Misal: namespace="category", allowList=["sepatu"]
//
//	→ datapoint ini akan ke-filter kalau search dengan filter category=sepatu
type DatapointRestrict struct {
	Namespace string   // nama field, misal "category", "brand"
	AllowList []string // nilai yang valid, misal ["sepatu", "tas"]
}

// SaveVector simpan/update 1 vector ke index
// Pakai ini untuk produk baru atau update produk existing
func (v *VectorSearchClient) SaveVector(
	ctx context.Context,
	id string,
	vector []float32,
) error {
	return v.SaveVectors(ctx, []Datapoint{
		{ID: id, Vector: vector},
	})
}

// SaveVectorWithMetadata simpan vector beserta metadata untuk filtering
func (v *VectorSearchClient) SaveVectorWithMetadata(
	ctx context.Context,
	id string,
	vector []float32,
	metadata map[string][]string, // misal {"category": ["sepatu"], "brand": ["Nike"]}
) error {
	restricts := make([]DatapointRestrict, 0, len(metadata))
	for ns, values := range metadata {
		restricts = append(restricts, DatapointRestrict{
			Namespace: ns,
			AllowList: values,
		})
	}

	return v.SaveVectors(ctx, []Datapoint{
		{
			ID:        id,
			Vector:    vector,
			Restricts: restricts,
		},
	})
}

// SaveVectors batch save banyak vector sekaligus (lebih efisien)
// Max ~100 vector per call (cek quota Vertex AI)
func (v *VectorSearchClient) SaveVectors(ctx context.Context, datapoints []Datapoint) error {
	if len(datapoints) == 0 {
		return nil
	}

	// Convert ke proto format yang dipahami Vertex AI API
	pbDatapoints := make([]*aiplatformpb.IndexDatapoint, len(datapoints))
	for i, dp := range datapoints {
		// Build restricts (metadata filter)
		restricts := make([]*aiplatformpb.IndexDatapoint_Restriction, 0, len(dp.Restricts))
		for _, r := range dp.Restricts {
			restricts = append(restricts, &aiplatformpb.IndexDatapoint_Restriction{
				Namespace: r.Namespace,
				AllowList: r.AllowList,
			})
		}

		pbDatapoints[i] = &aiplatformpb.IndexDatapoint{
			DatapointId:   dp.ID,
			FeatureVector: dp.Vector,
			Restricts:     restricts,
		}
	}

	// Kirim Upsert request
	// "Upsert" = Update kalau ID sudah ada, Insert kalau belum
	req := &aiplatformpb.UpsertDatapointsRequest{
		Index:      v.indexName,
		Datapoints: pbDatapoints,
	}

	_, err := v.indexClient.UpsertDatapoints(ctx, req)
	if err != nil {
		return fmt.Errorf("upsert datapoints: %w", err)
	}

	return nil
}

// DeleteVectors hapus vector berdasarkan ID
func (v *VectorSearchClient) DeleteVectors(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	req := &aiplatformpb.RemoveDatapointsRequest{
		Index:        v.indexName,
		DatapointIds: ids,
	}

	_, err := v.indexClient.RemoveDatapoints(ctx, req)
	if err != nil {
		return fmt.Errorf("remove datapoints: %w", err)
	}

	return nil
}

// ============================================================
// SEARCH OPERATIONS (Find Nearest Neighbors)
// ============================================================

// SearchResult hasil pencarian
type SearchResult struct {
	ID       string  // ID dari datapoint yang ditemukan
	Distance float64 // jarak ke query vector (semakin kecil = semakin mirip)
}

// SimilarityScore convert distance ke score 0-1 (semakin tinggi = semakin mirip)
// Untuk Cosine distance: similarity = 1 - distance/2
func (r SearchResult) SimilarityScore() float64 {
	score := 1.0 - float64(r.Distance)/2.0
	if score < 0 {
		score = 0
	}
	return score
}

// SearchFilter filter berdasarkan metadata saat search
type SearchFilter struct {
	Namespace string   // field yang mau di-filter, misal "category"
	AllowList []string // nilai yang BOLEH (harus match minimal 1)
	DenyList  []string // nilai yang TIDAK BOLEH
}

// Search cari topK vector paling mirip dengan query vector
func (v *VectorSearchClient) Search(
	ctx context.Context,
	queryVector []float32,
	topK int,
) ([]SearchResult, error) {
	return v.SearchWithFilters(ctx, queryVector, topK, nil)
}

// SearchWithFilters cari dengan filter metadata
// Contoh use case: cari sepatu mirip, tapi cuma brand Nike & Adidas
//
//	filters := []SearchFilter{
//	    {Namespace: "category", AllowList: []string{"sepatu"}},
//	    {Namespace: "brand", AllowList: []string{"Nike", "Adidas"}},
//	}
func (v *VectorSearchClient) SearchWithFilters(
	ctx context.Context,
	queryVector []float32,
	topK int,
	filters []SearchFilter,
) ([]SearchResult, error) {
	// Build restricts dari filters
	restricts := make([]*aiplatformpb.IndexDatapoint_Restriction, 0, len(filters))
	for _, f := range filters {
		restricts = append(restricts, &aiplatformpb.IndexDatapoint_Restriction{
			Namespace: f.Namespace,
			AllowList: f.AllowList,
			DenyList:  f.DenyList,
		})
	}

	// Build query datapoint
	// "Datapoint" di sini adalah vector yang mau dicari kemiripannya
	queryDatapoint := &aiplatformpb.IndexDatapoint{
		FeatureVector: queryVector,
		Restricts:     restricts,
	}

	// Build request
	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint:   v.indexEndpointName,
		DeployedIndexId: v.deployedIndexID,
		Queries: []*aiplatformpb.FindNeighborsRequest_Query{
			{
				Datapoint:     queryDatapoint,
				NeighborCount: int32(topK),
			},
		},
		// false = cuma return ID & distance (cepat)
		// true  = return juga vector lengkapnya (lebih lambat, jarang dibutuhkan)
		ReturnFullDatapoint: false,
	}

	// Kirim request ke Match API
	resp, err := v.matchClient.FindNeighbors(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("find neighbors: %w", err)
	}

	// Parse response
	// resp.NearestNeighbors[0] karena kita cuma kirim 1 query
	// Untuk batch search, bisa kirim multiple queries dalam 1 request
	if len(resp.NearestNeighbors) == 0 {
		return []SearchResult{}, nil
	}

	neighbors := resp.NearestNeighbors[0].Neighbors
	results := make([]SearchResult, len(neighbors))
	for i, n := range neighbors {
		results[i] = SearchResult{
			ID:       n.Datapoint.DatapointId,
			Distance: n.Distance,
		}
	}

	return results, nil
}

// BatchSearch cari multiple queries sekaligus (lebih efisien dari loop)
// Berguna untuk: rekomendasi based on multiple items, A/B testing query
func (v *VectorSearchClient) BatchSearch(
	ctx context.Context,
	queryVectors [][]float32,
	topK int,
) ([][]SearchResult, error) {
	queries := make([]*aiplatformpb.FindNeighborsRequest_Query, len(queryVectors))
	for i, qv := range queryVectors {
		queries[i] = &aiplatformpb.FindNeighborsRequest_Query{
			Datapoint: &aiplatformpb.IndexDatapoint{
				FeatureVector: qv,
			},
			NeighborCount: int32(topK),
		}
	}

	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint:   v.indexEndpointName,
		DeployedIndexId: v.deployedIndexID,
		Queries:         queries,
	}

	resp, err := v.matchClient.FindNeighbors(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("batch find neighbors: %w", err)
	}

	// Parse - 1 array hasil per query
	results := make([][]SearchResult, len(resp.NearestNeighbors))
	for i, nn := range resp.NearestNeighbors {
		queryResults := make([]SearchResult, len(nn.Neighbors))
		for j, n := range nn.Neighbors {
			queryResults[j] = SearchResult{
				ID:       n.Datapoint.DatapointId,
				Distance: n.Distance,
			}
		}
		results[i] = queryResults
	}

	return results, nil
}
