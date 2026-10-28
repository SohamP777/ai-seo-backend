// pkg/shopify/graphql.go - REAL GraphQL Mutations
package shopify

import (
    "context"
    "fmt"
)

// REAL Product Update using GraphQL
func (c *ShopifyClient) UpdateProductGraphQL(ctx context.Context, product *ShopifyProduct) error {
    query := `
    mutation productUpdate($input: ProductInput!) {
        productUpdate(input: $input) {
            product {
                id
                title
                descriptionHtml
                handle
                metafields(first: 10) {
                    edges {
                        node {
                            namespace
                            key
                            value
                        }
                    }
                }
                seo {
                    title
                    description
                }
            }
            userErrors {
                field
                message
            }
        }
    }`
    
    variables := map[string]interface{}{
        "input": map[string]interface{}{
            "id":            fmt.Sprintf("gid://shopify/Product/%d", product.ID),
            "title":         product.Title,
            "descriptionHtml": product.BodyHTML,
            "handle":        product.Handle,
            "status":        product.Status,
            "seo": map[string]interface{}{
                "title":       product.SEO.Title,
                "description": product.SEO.Description,
            },
        },
    }
    
    var response struct {
        Data struct {
            ProductUpdate struct {
                Product struct {
                    ID          string `json:"id"`
                    Title       string `json:"title"`
                    DescriptionHtml string `json:"descriptionHtml"`
                    Handle      string `json:"handle"`
                    Metafields  struct {
                        Edges []struct {
                            Node Metafield `json:"node"`
                        } `json:"edges"`
                    } `json:"metafields"`
                    SEO struct {
                        Title       string `json:"title"`
                        Description string `json:"description"`
                    } `json:"seo"`
                } `json:"product"`
                UserErrors []struct {
                    Field   string   `json:"field"`
                    Message string   `json:"message"`
                } `json:"userErrors"`
            } `json:"productUpdate"`
        } `json:"data"`
    }
    
    if err := c.GraphQLRequest(ctx, query, variables, &response); err != nil {
        return err
    }
    
    if len(response.Data.ProductUpdate.UserErrors) > 0 {
        return fmt.Errorf("graphql errors: %v", response.Data.ProductUpdate.UserErrors)
    }
    
    return nil
}

// REAL Metafields Set using GraphQL (more efficient than REST)
func (c *ShopifyClient) SetMetafieldsGraphQL(ctx context.Context, ownerID string, metafields []Metafield) error {
    query := `
    mutation metafieldsSet($metafields: [MetafieldsSetInput!]!) {
        metafieldsSet(metafields: $metafields) {
            metafields {
                id
                namespace
                key
                value
            }
            userErrors {
                field
                message
            }
        }
    }`
    
    var metafieldsInput []map[string]interface{}
    for _, mf := range metafields {
        metafieldsInput = append(metafieldsInput, map[string]interface{}{
            "ownerId":     ownerID,
            "namespace":   mf.Namespace,
            "key":         mf.Key,
            "value":       mf.Value,
            "type":        mf.Type,
            "description": mf.Description,
        })
    }
    
    variables := map[string]interface{}{
        "metafields": metafieldsInput,
    }
    
    var response struct {
        Data struct {
            MetafieldsSet struct {
                Metafields []Metafield `json:"metafields"`
                UserErrors []struct {
                    Field   string `json:"field"`
                    Message string `json:"message"`
                } `json:"userErrors"`
            } `json:"metafieldsSet"`
        } `json:"data"`
    }
    
    if err := c.GraphQLRequest(ctx, query, variables, &response); err != nil {
        return err
    }
    
    if len(response.Data.MetafieldsSet.UserErrors) > 0 {
        return fmt.Errorf("graphql errors: %v", response.Data.MetafieldsSet.UserErrors)
    }
    
    return nil
}

// REAL Bulk Operation for large stores
func (c *ShopifyClient) CreateBulkOperation(ctx context.Context, query string) (string, error) {
    mutation := `
    mutation bulkOperationRunQuery($query: String!) {
        bulkOperationRunQuery(query: $query) {
            bulkOperation {
                id
                status
            }
            userErrors {
                field
                message
            }
        }
    }`
    
    variables := map[string]interface{}{
        "query": query,
    }
    
    var response struct {
        Data struct {
            BulkOperationRunQuery struct {
                BulkOperation struct {
                    ID     string `json:"id"`
                    Status string `json:"status"`
                } `json:"bulkOperation"`
                UserErrors []struct {
                    Field   string `json:"field"`
                    Message string `json:"message"`
                } `json:"userErrors"`
            } `json:"bulkOperationRunQuery"`
        } `json:"data"`
    }
    
    if err := c.GraphQLRequest(ctx, mutation, variables, &response); err != nil {
        return "", err
    }
    
    if len(response.Data.BulkOperationRunQuery.UserErrors) > 0 {
        return "", fmt.Errorf("bulk operation errors: %v", response.Data.BulkOperationRunQuery.UserErrors)
    }
    
    return response.Data.BulkOperationRunQuery.BulkOperation.ID, nil
}