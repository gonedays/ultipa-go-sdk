package api

import (
	"log"
	"sync"
	ultipa "ultipa-go-sdk/rpc"
	"ultipa-go-sdk/sdk/configuration"
	"ultipa-go-sdk/sdk/structs"
)

func (api *UltipaAPI) InsertEdgesBatch(table *ultipa.EdgeTable, config *configuration.RequestConfig) (*ultipa.InsertEdgesReply, error) {
	client, conf, err := api.GetClient(config)

	if err != nil {
		return nil, err
	}

	ctx, _ := api.Pool.NewContext(config)

	resp, err := client.InsertEdges(ctx, &ultipa.InsertEdgesRequest{
		GraphName: conf.CurrentGraph,
		EdgeTable: table,
		Silent:    true,
	})

	return resp, err
}

func (api *UltipaAPI) InsertEdgesBatchBySchema(schema *structs.Schema, rows []*structs.Edge, config *configuration.RequestConfig) (*ultipa.InsertEdgesReply, error) {
	client, conf, err := api.GetClient(config)

	if err != nil {
		return nil, err
	}

	ctx, _ := api.Pool.NewContext(config)

	table := &ultipa.EdgeTable{}

	table.Schemas = []*ultipa.Schema{
		{
			SchemaName: schema.Name,
			Properties: []*ultipa.Property{},
		},
	}

	for _, prop := range schema.Properties {

		if prop.IsIDType() || prop.IsIgnore() {
			continue
		}

		table.Schemas[0].Properties = append(table.Schemas[0].Properties, &ultipa.Property{
			PropertyName: prop.Name,
			PropertyType: prop.Type,
		})
	}

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	for _, row := range rows {

		wg.Add(1)

		go func(row *structs.Edge) {
			defer wg.Done()

			newnode := &ultipa.EdgeRow{
				FromId:     row.From,
				ToId:       row.To,
				SchemaName: schema.Name,
			}

			for _, prop := range schema.Properties {

				if prop.IsIDType() || prop.IsIgnore() {
					continue
				}

				bs, err := row.GetBytes(prop.Name)

				if err != nil {
					log.Fatal("Get row bytes value failed ", prop.Name, " ",err)
				}

				newnode.Values = append(newnode.Values, bs)
			}

			mtx.Lock()
			table.EdgeRows = append(table.EdgeRows, newnode)
			mtx.Unlock()
		}(row)
	}

	wg.Wait()

	resp, err := client.InsertEdges(ctx, &ultipa.InsertEdgesRequest{
		GraphName:  conf.CurrentGraph,
		EdgeTable:  table,
		InsertType: ultipa.InsertType_OVERWRITE,
		Silent:     true,
	})

	return resp, err
}
