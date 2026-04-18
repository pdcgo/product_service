package product

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/product_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
)

// ProductListExport implements [product_ifaceconnect.ProductServiceHandler].
func (p *productSrvImpl) ProductListExport(
	ctx context.Context,
	req *connect.Request[product_iface.ProductListExportRequest],
	stream *connect.ServerStream[product_iface.ProductListExportResponse],
) error {
	var err error
	streamlog := func(msg string) {
		stream.Send(&product_iface.ProductListExportResponse{
			Message: msg,
		})
	}

	streamerr := func(err error) error {
		stream.Send(&product_iface.ProductListExportResponse{
			Message: err.Error(),
		})

		return err
	}

	pay := req.Msg
	sort := req.Msg.Sort

	db := p.db.WithContext(ctx)

	// with d as (
	// 	select
	// 		oi.product_id,
	// 		sum(oi.total) as revenue

	// 	from public.order_items oi
	// 	join public.orders o on o.id = oi.order_id
	// 	where
	// 		o.team_id = 70
	// 	group by oi.product_id
	// )

	// select
	// 	p.*
	// from d
	// left join public.products p on p.id = d.product_id
	trange := req.Msg.TimeRange

	var tstart, tend time.Time
	if trange.EndDate.IsValid() {
		tend = trange.EndDate.AsTime()
	} else {
		tend = time.Now()
	}

	if trange.StartDate.IsValid() {
		tstart = trange.StartDate.AsTime()
	} else {
		tstart = tend.AddDate(0, 0, -14)
	}

	squery := db.
		Table("public.order_items oi").
		Joins("left join public.orders o on o.id = oi.order_id").
		Select([]string{
			"oi.product_id",
			"count(distinct o.team_id) as team_count",
			"count(distinct o.order_mp_id) as shop_count",
			"count(distinct oi.order_id) as order_count",
			"count(oi.count) as piece_count",
			"sum(oi.total) as revenue_amount",
		}).
		Where("o.status != ?", db_models.OrdCancel).
		Where("o.created_at between ? and ?", tstart, tend)

	switch soldby := pay.SoldBy.(type) {
	case *product_iface.ProductListExportRequest_SoldByTeamId:
		squery = squery.Where("o.team_id = ?", soldby.SoldByTeamId)
	case *product_iface.ProductListExportRequest_SoldByShopId:
		squery = squery.Where("o.order_mp_id = ?", soldby.SoldByShopId)
	}

	pquery := db.
		Table("public.products p").
		Joins("left join public.teams t on t.id = p.team_id").
		Joins(
			"join (?) d on p.id = d.product_id",
			squery.
				Group("oi.product_id"),
		).
		Where("p.deleted != true").
		Select([]string{
			"d.*",
			"p.name as name",
			"p.ref_id as ref_id",
			"p.stock_ready as ready_stock",
			"p.stock_pending as ongoing_stock",
			"p.stock_reserved as reserved_stock",
			"p.stock_ready + p.stock_pending as total_stock",
			"t.name as owner_team_name",
		})

	switch search := pay.Search.(type) {
	case *product_iface.ProductListExportRequest_ProductName:
		keyword := strings.ToLower(search.ProductName)
		pquery = pquery.Where("lower(p.name) like ?", "%"+keyword+"%")
	case *product_iface.ProductListExportRequest_SkuId:
		keyword := strings.ToLower(search.SkuId)
		pquery = pquery.Where("lower(p.ref_id) like ?", "%"+keyword+"%")
	}

	if pay.TeamId != 0 {
		pquery = pquery.Where("p.team_id = ?", pay.TeamId)
	}

	if pay.IsLocked {
		pquery = pquery.Where("p.cross_locked = ?", pay.IsLocked)
	}

	var total int64
	streamlog("counting data...")
	err = pquery.Session(&gorm.Session{}).Count(&total).Error
	if err != nil {
		return streamerr(err)
	}

	var field string

	switch sort.Field {
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_PIECE_COUNT:
		field = "piece_count"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_REVENUE_AMOUNT:
		field = "revenue_amount"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_ORDER_COUNT:
		field = "order_count"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_SHOP_COUNT:
		field = "shop_count"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_TEAM_COUNT:
		field = "team_count"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_ONGOING_STOCK:
		field = "ongoing_stock"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_READY_STOCK:
		field = "ready_stock"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_RESERVED_STOCK:
		field = "reserved_stock"
	case product_iface.ProductListFieldSort_PRODUCT_LIST_FIELD_SORT_TOTAL_STOCK:
		field = "total_stock"
	default:
		field = "order_count"
	}

	switch sort.Type {
	case common.SortType_SORT_TYPE_ASC:
		field = field + " asc nulls last"
	case common.SortType_SORT_TYPE_DESC:
		field = field + " desc nulls last"
	default:
		field = field + " desc nulls last"
	}

	streamlog("getting data...")
	writer := &connectStreamWriter{
		stream: stream,
		offset: 0,
		total:  total,
	}

	// debug multiwriter
	// f, err := os.OpenFile("product_list_export.csv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	// if err != nil {
	// 	return streamerr(err)
	// }
	// defer f.Close()
	// multiWriter := io.MultiWriter(writer, f)

	// csvWriter := csv.NewWriter(multiWriter)
	csvWriter := csv.NewWriter(writer)

	headers := []string{
		"Product Name",
		"SKU ID",
		"Owner Team",
		"Jumlah Toko",
		"Jumlah Team",
		"Jumlah Order",
		"Jumlah Piece",
		"Total Revenue",
		"Ongoing Stock",
		"Ready Stock",
		"Total Stock",
		"Min. Lock",
	}

	csvWriter.Write(headers)

	streamlog("writing data..")
	rows, err := pquery.
		Order(field).
		Rows()

	if err != nil {
		return streamerr(err)
	}
	defer rows.Close()

	for rows.Next() {

		select {
		case <-ctx.Done():
			return nil
		default:
			var p product_iface.ProductItem
			err = db.ScanRows(rows, &p)
			if err != nil {
				return streamerr(err)
			}

			err = csvWriter.Write([]string{
				p.Name,
				p.RefId,
				p.OwnerTeamName,
				fmt.Sprintf("%d", p.ShopCount),
				fmt.Sprintf("%d", p.TeamCount),
				fmt.Sprintf("%d", p.OrderCount),
				fmt.Sprintf("%d", p.PieceCount),
				fmt.Sprintf("%4.f", p.RevenueAmount),
				fmt.Sprintf("%d", p.OngoingStock),
				fmt.Sprintf("%d", p.ReadyStock),
				fmt.Sprintf("%d", p.OngoingStock+p.ReadyStock),
				fmt.Sprintf("%d", p.ReservedStock),
			})

			if err != nil {
				return streamerr(err)
			}

			if writer.c%50 == 0 {
				csvWriter.Flush()
			}

		}

	}

	csvWriter.Flush()
	return err

}

type connectStreamWriter struct {
	stream *connect.ServerStream[product_iface.ProductListExportResponse]
	c      int
	offset int64
	total  int64
}

// Write implements io.Writer.
func (c *connectStreamWriter) Write(p []byte) (n int, err error) {
	c.c += len(p)
	c.offset += 1
	err = c.stream.Send(&product_iface.ProductListExportResponse{
		Offset: c.offset,
		Total:  c.total,
		Data:   p,
	})

	return len(p), err
}
