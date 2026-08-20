// pkg/shopify/theme_fixer.go
package shopify

import (
    "context"
    "fmt"
    "strings"
    "time"
    "strconv"
)

type ThemeFixer struct {
    client    *ShopifyClient
    backupMgr BackupManager
    themeID   string
    theme     string
    dryRun    bool
    results   []FixResult
}

func NewThemeFixer(client *ShopifyClient, backupMgr BackupManager, themeID int64, dryRun bool) *ThemeFixer {
    return &ThemeFixer{
        client:    client,
        backupMgr: backupMgr,
        themeID:   fmt.Sprintf("%d", themeID),
        dryRun:    dryRun,
        results:   []FixResult{},
    }
}

func (t *ThemeFixer) FixTheme(ctx context.Context) ([]FixResult, error) {
    // Get theme.liquid
    themeIDStr := t.themeID
    asset, err := t.client.GetThemeAsset(themeIDStr, "templates/theme.liquid")
    if err != nil {
        return nil, fmt.Errorf("get theme.liquid: %w", err)
    }

    // Get other template files (these return strings)
    productTemplate, _ := t.client.GetThemeAsset(themeIDStr, "templates/product.liquid")
    collectionTemplate, _ := t.client.GetThemeAsset(themeIDStr, "templates/collection.liquid")
    articleTemplate, _ := t.client.GetThemeAsset(themeIDStr, "templates/article.liquid")

    // Create ThemeAsset objects from the string returns
    assetObj := &ThemeAsset{
        Key:   "templates/theme.liquid",
        Value: asset,
    }
    
    productAssetObj := &ThemeAsset{
        Key:   "templates/product.liquid",
        Value: productTemplate,
    }
    
    collectionAssetObj := &ThemeAsset{
        Key:   "templates/collection.liquid",
        Value: collectionTemplate,
    }
    
    articleAssetObj := &ThemeAsset{
        Key:   "templates/article.liquid",
        Value: articleTemplate,
    }

    // Fix 1: Add meta tags to theme.liquid
    if result := t.addMetaTags(assetObj); result != nil {
        t.results = append(t.results, *result)
    }

    // Fix 2: Add JSON-LD schemas
    if result := t.addSchemas(assetObj.Value, productAssetObj.Value, collectionAssetObj.Value, articleAssetObj.Value); result != nil {
        t.results = append(t.results, *result)
    }

    // Fix 3: Add breadcrumb schema
    if result := t.addBreadcrumbSchema(productAssetObj.Value, collectionAssetObj.Value); result != nil {
        t.results = append(t.results, *result)
    }

    // Fix 4: Add Open Graph tags
    if result := t.addOpenGraphTags(assetObj); result != nil {
        t.results = append(t.results, *result)
    }

    // Fix 5: Add Twitter Card tags
    if result := t.addTwitterCardTags(assetObj); result != nil {
        t.results = append(t.results, *result)
    }

    // Fix 6: Add resource hints for performance
    if result := t.addResourceHints(assetObj); result != nil {
        t.results = append(t.results, *result)
    }

    // Apply changes if not dry run
    if !t.dryRun && len(t.results) > 0 {
        themeIDInt, err := strconv.ParseInt(t.themeID, 10, 64)
        if err != nil {
            return t.results, fmt.Errorf("invalid theme ID: %w", err)
        }

        if assetObj.Value != asset {
            if err := t.client.UpdateThemeAsset(ctx, themeIDInt, *assetObj); err != nil {
                return t.results, fmt.Errorf("update theme.liquid: %w", err)
            }
        }

        if productAssetObj.Value != productTemplate && productAssetObj.Value != "" {
            t.client.UpdateThemeAsset(ctx, themeIDInt, *productAssetObj)
        }
        if collectionAssetObj.Value != collectionTemplate && collectionAssetObj.Value != "" {
            t.client.UpdateThemeAsset(ctx, themeIDInt, *collectionAssetObj)
        }
        if articleAssetObj.Value != articleTemplate && articleAssetObj.Value != "" {
            t.client.UpdateThemeAsset(ctx, themeIDInt, *articleAssetObj)
        }
    }

    return t.results, nil
}

func (t *ThemeFixer) addMetaTags(asset *ThemeAsset) *FixResult {
    original := asset.Value

    metaTags := `
<!-- SEO Auto-Fixer: Added meta tags -->
<meta charset="utf-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="viewport" content="width=device-width,initial-scale=1,shrink-to-fit=no">
<link rel="canonical" href="{{ canonical_url }}">

{%- if page_description -%}
  <meta name="description" content="{{ page_description | escape }}">
{%- endif -%}

{%- if template contains 'product' -%}
  <meta property="og:title" content="{{ product.title | escape }}">
  <meta property="og:description" content="{{ product.description | strip_html | truncate: 200 | escape }}">
  <meta property="og:image" content="https:{{ product.featured_image | img_url: 'master' }}">
  <meta property="og:image:secure_url" content="https:{{ product.featured_image | img_url: 'master' }}">
  <meta property="og:image:width" content="{{ product.featured_image.width }}">
  <meta property="og:image:height" content="{{ product.featured_image.height }}">
  <meta property="og:type" content="product">
  <meta property="og:availability" content="{% if product.available %}instock{% else %}outofstock{% endif %}">
  <meta property="og:price:amount" content="{{ product.selected_or_first_available_variant.price | money_without_currency | remove: ',' }}">
  <meta property="og:price:currency" content="{{ shop.currency }}">
  <meta property="product:brand" content="{{ product.vendor | escape }}">
  <meta property="product:retailer_item_id" content="{{ product.selected_or_first_available_variant.sku }}">
{%- elsif template contains 'article' -%}
  <meta property="og:title" content="{{ article.title | escape }}">
  <meta property="og:description" content="{{ article.excerpt_or_content | strip_html | truncate: 200 | escape }}">
  <meta property="og:image" content="https:{{ article.image | img_url: 'master' }}">
  <meta property="og:image:secure_url" content="https:{{ article.image | img_url: 'master' }}">
  <meta property="og:type" content="article">
  <meta property="article:published_time" content="{{ article.published_at | date: '%Y-%m-%dT%H:%M:%S%z' }}">
  <meta property="article:author" content="{{ article.author | escape }}">
{%- elsif template contains 'collection' -%}
  <meta property="og:title" content="{{ collection.title | escape }}">
  <meta property="og:description" content="{{ collection.description | strip_html | truncate: 200 | escape }}">
  <meta property="og:image" content="https:{{ collection.image | img_url: 'master' }}">
  <meta property="og:image:secure_url" content="https:{{ collection.image | img_url: 'master' }}">
  <meta property="og:type" content="product.group">
{%- else -%}
  <meta property="og:title" content="{{ page_title | escape }}">
  <meta property="og:description" content="{{ page_description | default: shop.description | escape }}">
  <meta property="og:image" content="https:{{ settings.social_share_image | img_url: 'master' }}">
  <meta property="og:image:secure_url" content="https:{{ settings.social_share_image | img_url: 'master' }}">
  <meta property="og:type" content="website">
{%- endif -%}

<meta property="og:url" content="{{ canonical_url }}">
<meta property="og:site_name" content="{{ shop.name }}">`

    // Insert after <head> tag
    if strings.Contains(original, "<head>") {
        asset.Value = strings.Replace(original, "<head>", "<head>\n"+metaTags, 1)
    } else if strings.Contains(original, "<head") {
        asset.Value = strings.Replace(original, "<head", "<head>\n"+metaTags, 1)
    } else {
        asset.Value = metaTags + "\n" + original
    }

    if asset.Value != original {
        return &FixResult{
            Action:    "add_meta_tags",
            Target:    "theme.liquid",
            Before:    "No meta tags",
            After:     "Meta tags added",
            Message:   "Added SEO meta tags to theme",
            Timestamp: time.Now(),
        }
    }

    return nil
}

func (t *ThemeFixer) addOpenGraphTags(asset *ThemeAsset) *FixResult {
    // Already added in meta tags
    return nil
}

func (t *ThemeFixer) addSchemas(themeLiquid, productTemplate, collectionTemplate, articleTemplate string) *FixResult {
    // implementation
    return nil
}

func (t *ThemeFixer) addBreadcrumbSchema(productTemplate, collectionTemplate string) *FixResult {
    // implementation
    return nil
}

func (t *ThemeFixer) addTwitterCardTags(asset *ThemeAsset) *FixResult {
    original := asset.Value

    twitterTags := `
<!-- Twitter Card tags -->
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{ page_title | escape }}">
<meta name="twitter:description" content="{{ page_description | default: shop.description | escape }}">
<meta name="twitter:image" content="https:{{ settings.social_share_image | img_url: 'master' }}">
<meta name="twitter:site" content="@{{ settings.twitter_handle }}">`

    // Insert near meta tags
    if !strings.Contains(original, "twitter:card") {
        if strings.Contains(original, "<!-- SEO Auto-Fixer") {
            asset.Value = strings.Replace(original, "<!-- SEO Auto-Fixer", twitterTags+"\n<!-- SEO Auto-Fixer", 1)
        } else {
            asset.Value = original + "\n" + twitterTags
        }

        if asset.Value != original {
            return &FixResult{
                Action:    "add_twitter_cards",
                Target:    "theme.liquid",
                Before:    "No Twitter cards",
                After:     "Twitter cards added",
                Message:   "Added Twitter Card meta tags",
                Timestamp: time.Now(),
            }
        }
    }

    return nil
}

func (t *ThemeFixer) addResourceHints(asset *ThemeAsset) *FixResult {
    original := asset.Value

    resourceHints := `
<!-- Resource hints for performance -->
<link rel="dns-prefetch" href="https://cdn.shopify.com">
<link rel="dns-prefetch" href="https://shop.app">
<link rel="preconnect" href="https://cdn.shopify.com" crossorigin>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="preload" as="style" href="{{ 'theme.css' | asset_url }}">
<link rel="preload" as="script" href="{{ 'theme.js' | asset_url }}">`

    if !strings.Contains(original, "resource hints") {
        if strings.Contains(original, "<!-- SEO Auto-Fixer") {
            asset.Value = strings.Replace(original, "<!-- SEO Auto-Fixer", resourceHints+"\n<!-- SEO Auto-Fixer", 1)
        } else if strings.Contains(original, "<head>") {
            asset.Value = strings.Replace(original, "<head>", "<head>\n"+resourceHints, 1)
        }

        if asset.Value != original {
            return &FixResult{
                Action:    "add_resource_hints",
                Target:    "theme.liquid",
                Before:    "No resource hints",
                After:     "Resource hints added",
                Message:   "Added performance resource hints",
                Timestamp: time.Now(),
            }
        }
    }

    return nil
}