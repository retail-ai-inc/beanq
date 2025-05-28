package routers

import (
	"math"
	"net/http"

	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WorkFlow struct {
	workflowCollection *mongo.Collection
}

func NewWorkFlow(collection *mongo.Collection) *WorkFlow {
	return &WorkFlow{workflowCollection: collection}
}

func (t *WorkFlow) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	query := r.URL.Query()
	page := cast.ToInt64(query.Get("page"))
	pageSize := cast.ToInt64(query.Get("pageSize"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	ctx := r.Context()

	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "CreatedAt", Value: -1}})

	filter := bson.M{}
	cursor, err := t.workflowCollection.Find(ctx, filter, opts)
	defer func() {
		_ = cursor.Close(ctx)
	}()
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	var data []bson.M
	if err := cursor.All(ctx, &data); err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	total, err := t.workflowCollection.CountDocuments(ctx, filter)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	datas := make(map[string]any, 3)
	datas["data"] = data
	datas["total"] = math.Ceil(float64(total) / float64(pageSize))
	datas["cursor"] = page
	result.Data = datas
	_ = result.Json(w, http.StatusOK)
}

func (t *WorkFlow) Delete(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	ctx := r.Context()
	id := r.PostFormValue("id")

	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			res.Code = berror.InternalServerErrorCode
			res.Msg = err.Error()
			_ = res.Json(w, http.StatusInternalServerError)
			return
		}
		filter["_id"] = nid
	}

	result, err := t.workflowCollection.DeleteOne(ctx, filter)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	res.Data = result.DeletedCount
	_ = res.Json(w, http.StatusOK)
}
