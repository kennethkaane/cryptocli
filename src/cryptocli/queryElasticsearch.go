package main

//import (
//	"time"
//	"strconv"
//	"encoding/json"
//	"log"
//	"github.com/tehmoon/errors"
//	"github.com/spf13/pflag"
//	"sync"
//	"github.com/olivere/elastic"
//	"io"
//	"context"
//)
//
//func init() {
//	MODULELIST.Register("query-elasticsearch", "Send query to elasticsearch cluster and output result in json line", NewQueryElasticsearch)
//}
//
//func (m *QueryElasticsearch) SetFlagSet(fs *pflag.FlagSet, args []string) {
//	m.flags = &QueryElasticsearchFlags{}
//
//	fs.StringVar(&m.flags.From, "from", "now-15m", "Elasticsearch date for gte")
//	fs.StringVar(&m.flags.To, "to", "now", "Elasticsearch date for lt. Has not effect when \"--tail\" is used")
//	fs.BoolVar(&m.flags.Asc, "asc", false, "Sort by asc")
//	fs.StringVar(&m.flags.Sort, "sort", "@timestamp", "Sort field")
//	fs.StringVar(&m.flags.TimestampField, "timestamp-field", "@timestamp", "Timestamp field")
//	fs.IntVar(&m.flags.Size, "size", 0, "Overall number of results to display, does not change the scroll size")
//	fs.IntVar(&m.flags.ScrollSize, "scroll-size", 500, "Document to return between each scroll")
//	fs.StringVar(&m.flags.QueryStringQuery, "query", "*", "Elasticsearch query string query")
//	fs.StringVar(&m.flags.Server, "server", "http://localhost:9200", "Specify elasticsearch server to query")
//	fs.StringVar(&m.flags.Index, "index", "", "Specify the elasticsearch index to query")
//	fs.BoolVar(&m.flags.CountOnly, "count-only", false, "Only displays the match number")
//	fs.StringVar(&m.flags.Aggregation, "aggregation", "", "Elastic Aggregation query")
//	fs.BoolVar(&m.flags.Tail, "tail", false, "Query Elasticsearch in tail -f style. Deactivate the flag \"--to\"")
//	fs.StringArrayVar(&m.flags.SortFields, "sort-field", make([]string, 0), "Additional fields to sort on")
//	fs.DurationVar(&m.flags.TailInterval, "tail-interval", time.Second, "Time to wait before querying elasticsearch again when using \"--tail\"")
//	fs.DurationVar(&m.flags.TailMax, "tail-max", time.Duration((1 << 63) - 1), "Maximum time to wait before exiting the \"--tail\" loop")
//}
//
//func (m *QueryElasticsearch) Init(in, out chan *Message, global *GlobalFlags) (err error) {
//	if m.flags.TailInterval < 1 {
//		return errors.Errorf("Flag %q cannot be lower than 1", "--tail-interval")
//	}
//
//	if m.flags.Index == "" {
//		return errors.Errorf("Flag %q is required", "--index")
//	}
//
//	if m.flags.TimestampField == "" {
//		return errors.Errorf("Flag %q cannot be empty", "--timestamp-field")
//	}
//
//	if m.flags.To == "" {
//		return errors.Errorf("Flag %q cannot be empty", "--to")
//	}
//
//	if m.flags.From == "" {
//		return errors.Errorf("Flag %q cannot be empty", "--from")
//	}
//
//	if m.flags.Size < 0 {
//		return errors.Errorf("Flag %q cannot be negative", "--size")
//	}
//
//	if m.flags.ScrollSize < 1 {
//		return errors.Errorf("Flag %q cannot be less than 1", "--scroll-size")
//	}
//
//	if m.flags.ScrollSize > m.flags.Size && m.flags.Size != 0 {
//		m.flags.ScrollSize = m.flags.Size
//	}
//
//	setURL := elastic.SetURL(m.flags.Server)
//
//	m.client, err = elastic.NewClient(setURL, elastic.SetSniff(false))
//	if err != nil {
//		return errors.Wrapf(err, "Err creating connection to server %s", m.flags.Server)
//	}
//
//	go func(m *QueryElasticsearch, in, out chan *Message) {
//		wg := &sync.WaitGroup{}
//		ctx, cancel := context.WithCancel(context.Background())
//		outc := make(MessageChannel)
//
//		out <- &Message{
//			Type: MessageTypeChannel,
//			Interface: outc,
//		}
//
//		LOOP: for message := range in {
//			switch message.Type {
//				case MessageTypeTerminate:
//					wg.Wait()
//					cancel()
//					out <- message
//					break LOOP
//				case MessageTypeChannel:
//					inc, ok := message.Interface.(MessageChannel)
//					if ok {
//						wg.Add(2)
//						go DrainChannel(inc, wg)
//						go func(m *QueryElasticsearch, outc MessageChannel, wg *sync.WaitGroup, ctx context.Context) {
//							defer wg.Done()
//							defer close(outc)
//
//							args := &QueryElasticsearchFuncArgs{
//								Client: m.client,
//								Flags: m.flags,
//								BoolQuery: QueryElasticsearchGenerateBoolQuery(m.flags, true),
//							}
//
//							ts, err := QueryElasticsearchDo(args, outc, ctx)
//							if err != nil {
//								log.Println(err.Error())
//								return
//							}
//
//							if args.Flags.Tail {
//								ctx, cancel = context.WithTimeout(ctx, m.flags.TailMax)
//								defer cancel()
//
//								timer := time.NewTimer(m.flags.TailInterval)
//								timer.Stop()
//
//								for {
//									args.Flags.From = ts
//									args.BoolQuery = QueryElasticsearchGenerateBoolQuery(m.flags, false)
//
//									timer.Reset(m.flags.TailInterval)
//									select {
//										case <- timer.C:
//										case <- ctx.Done():
//											log.Println("Timeout exceeded")
//											timer.Stop()
//											return
//									}
//
//									ts, err = QueryElasticsearchDo(args, outc, ctx)
//									if err != nil {
//										log.Println(err.Error())
//										return
//									}
//								}
//							}
//						}(m, outc, wg, ctx)
//						out <- &Message{
//							Type: MessageTypeTerminate,
//						}
//						break LOOP
//					}
//
//			}
//		}
//
//		wg.Wait()
//		cancel()
//		// Last message will signal the closing of the channel
//		<- in
//		close(out)
//	}(m, in, out)
//
//	return nil
//}
//
//type QueryElasticsearch struct {
//	fs *pflag.FlagSet
//	client *elastic.Client
//	flags *QueryElasticsearchFlags
//}
//
//type QueryElasticsearchFlags struct {
//	Version int
//	QueryStringQuery string
//	Server string
//	Index string
//	From string
//	To string
//	Size int
//	Asc bool
//	CountOnly bool
//	Sort string
//	ScrollSize int
//	TimestampField string
//	Aggregation string
//	Tail bool
//	SortFields []string
//	TailInterval time.Duration
//	TailMax time.Duration
//}
//
//func QueryElasticsearchGenerateBoolQuery(flags *QueryElasticsearchFlags, gte bool) (bq *elastic.BoolQuery) {
//		qs := elastic.NewQueryStringQuery(flags.QueryStringQuery)
//		rq := elastic.NewRangeQuery(flags.TimestampField)
//
//		if ! flags.Tail {
//			rq.Lt(flags.To)
//		}
//
//		if gte {
//			rq.Gte(flags.From)
//		} else {
//			rq.Gt(flags.From)
//		}
//
//		return elastic.NewBoolQuery().Must(qs, rq)
//}
//
//func QueryElasticsearchDo(args *QueryElasticsearchFuncArgs, outc MessageChannel, ctx context.Context) (ts string, err error) {
//	if args.Flags.Aggregation == "" {
//		ts, err = QueryElasticsearchDoSearch(args, outc, ctx)
//		if err != nil {
//			return ts, errors.Wrap(err, "Error in search")
//		}
//
//		return ts, nil
//	}
//
//	ts, err = QueryElasticsearchDoAggregation(args, outc, ctx)
//	if err != nil {
//		return ts, errors.Wrap(err, "Error in aggregation")
//	}
//
//	return ts, nil
//}
//
//func QueryElasticsearchParseTimestamp(field string, hits []*elastic.SearchHit, asc bool) (ts string) {
//	pos := 0
//	if asc {
//		pos = len(hits) - 1
//	}
//
//	payload := hits[pos].Source
//
//	var hit map[string]interface{}
//
//	err := json.Unmarshal(payload, &hit)
//	if err != nil {
//		err = errors.Wrap(err, "Un-expected unable to unmarshal source to json")
//		log.Println(err.Error())
//		return ""
//	}
//
//	if timestamp, found := hit[field]; found {
//		if timestamp, ok := timestamp.(string); ok {
//			return timestamp
//		}
//	}
//
//	return ""
//}
//
//func QueryElasticsearchDoSearch(args *QueryElasticsearchFuncArgs, outc MessageChannel, ctx context.Context) (ts string, err error) {
//	ts = args.Flags.From
//
//	scroll := args.Client.Scroll(args.Flags.Index).
//		Query(args.BoolQuery).
//		Sort(args.Flags.Sort, args.Flags.Asc).
//		Scroll("15s").
//		Size(args.Flags.ScrollSize)
//
//	for _, field := range args.Flags.SortFields {
//		scroll.Sort(field, args.Flags.Asc)
//	}
//
//	res, err := scroll.Do(ctx)
//	if err != nil {
//		if err != io.EOF {
//			return ts, errors.Wrap(err, "Err querying elasticsearch")
//		}
//	}
//
//	if res == nil || res.TotalHits() == 0 {
//		if args.Flags.CountOnly {
//			outc <- []byte("0\n")
//		}
//
//		return ts, nil
//	}
//
//	scrollId := res.ScrollId
//	defer QueryElasticsearchClearScroll(args.Client, scrollId)
//
//	ts = QueryElasticsearchParseTimestamp(args.Flags.TimestampField, res.Hits.Hits, args.Flags.Asc)
//
//	if args.Flags.CountOnly {
//		var totalHits int64 = 0
//
//		totalHits = res.TotalHits()
//
//		outc <- append([]byte(strconv.FormatInt(totalHits, 10)), '\n')
//
//		return ts, nil
//	}
//
//	counter := 0
//
//	for i := 0; (counter != args.Flags.Size || counter == 0); i++ {
//		if i == len(res.Hits.Hits) {
//			break
//		}
//
//		payload, err := json.Marshal(res.Hits.Hits[i])
//		if err != nil {
//			return ts, errors.Wrap(err, "Un-excepted error marshaling hit")
//		}
//
//		outc <- payload
//
//		counter++
//	}
//
//	if counter == args.Flags.Size && counter != 0 {
//		return ts, nil
//	}
//
//	LOOP: for {
//		res, err := args.Client.Scroll(args.Flags.Index).
//			Query(args.BoolQuery).
//			Scroll("15s").
//			ScrollId(scrollId).
//			Do(ctx)
//		if err != nil {
//			if err == io.EOF {
//				break LOOP
//			}
//
//			return ts, errors.Wrap(err, "Err querying elasticsearch")
//		}
//
//		if args.Flags.Asc {
//			ts = QueryElasticsearchParseTimestamp(args.Flags.TimestampField, res.Hits.Hits, args.Flags.Asc)
//		}
//
//		for i := 0; (counter != args.Flags.Size || counter == 0); i++ {
//			if i == len(res.Hits.Hits) {
//				break
//			}
//
//			payload, err := json.Marshal(res.Hits.Hits[i])
//			if err != nil {
//				return ts, errors.Wrap(err, "Un-excepted error marshaling hit")
//			}
//
//			outc <- payload
//
//			counter++
//		}
//
//		scrollId = res.ScrollId
//
//		if counter == args.Flags.Size && counter != 0 {
//			break LOOP
//		}
//	}
//
//	return ts, nil
//}
//
//func QueryElasticsearchClearScroll(client *elastic.Client, id string) (err error) {
//	_, err = client.ClearScroll(id).
//		Do(context.Background())
//	if err != nil {
//		return errors.Wrapf(err, "Failed to clear the scrollid %s", id)
//	}
//
//	return nil
//}
//
//type QueryElasticsearchFuncArgs struct {
//	Client *elastic.Client
//	Flags *QueryElasticsearchFlags
//	BoolQuery *elastic.BoolQuery
//}
//
//func NewQueryElasticsearch() (Module) {
//	return &QueryElasticsearch{}
//}
//
//type QueryElasticsearchStringAggregation struct{
//	body string
//}
//
//func (a QueryElasticsearchStringAggregation) Source() (v interface{}, err error) {
//	err = json.Unmarshal([]byte(a.body), &v)
//
//	return v, err
//}
//func QueryElasticsearchDoAggregation(args *QueryElasticsearchFuncArgs, outc MessageChannel, ctx context.Context) (ts string, err error) {
//	aggregation := &QueryElasticsearchStringAggregation{
//		body: args.Flags.Aggregation,
//	}
//
//	ts = args.Flags.From
//
//	res, err := args.Client.Search(args.Flags.Index).
//		Query(args.BoolQuery).
//		Size(1).
//		Sort(args.Flags.Sort, false).
//		Aggregation("root", aggregation).
//		Do(ctx)
//	if err != nil {
//		if err != io.EOF {
//			return ts, errors.Wrap(err, "Err querying elasticsearch")
//		}
//	}
//
//	if res == nil {
//		if args.Flags.CountOnly {
//			outc <- []byte("0\n")
//		}
//
//		return ts, nil
//	}
//
//	if len(res.Hits.Hits) == 1 {
//		ts = QueryElasticsearchParseTimestamp(args.Flags.TimestampField, res.Hits.Hits, args.Flags.Asc)
//	}
//
//	if res.Aggregations != nil {
//		if args.Flags.CountOnly {
//			outc <- []byte("0\n")
//
//			return ts, nil
//		}
//
//		payload, err := json.Marshal(res.Aggregations)
//		if err != nil {
//			return ts, errors.Wrap(err, "Aggregations results are empty")
//		}
//
//		outc <- payload
//	}
//
//	return ts, nil
//}
