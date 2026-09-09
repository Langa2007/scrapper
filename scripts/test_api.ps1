# Test script for AI Internet Scraper Bot API
$BaseUrl = "http://localhost:8080"

Write-Host "`n--- 1. Testing Health Endpoint ---" -ForegroundColor Cyan
try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/api/v1/health" -Method Get
    $health | ConvertTo-Json -Depth 4 | Write-Host -ForegroundColor Green
} catch {
    Write-Error "Health check failed: $_"
}

Write-Host "`n--- 2. Testing News Extraction (For Dira News integration) ---" -ForegroundColor Cyan
try {
    $newsBody = @{
        topic = "technology"
        limit = 3
        timeframe = "w"
    } | ConvertTo-Json
    $news = Invoke-RestMethod -Uri "$BaseUrl/api/v1/news" -Method Post -Body $newsBody -ContentType "application/json"
    Write-Host "Found $($news.count) articles for topic '$($news.topic)':" -ForegroundColor Green
    foreach ($art in $news.articles) {
        Write-Host "  * Title: $($art.title)" -ForegroundColor Yellow
        Write-Host "    Source: $($art.source) | URL: $($art.url)"
        Write-Host "    Summary: $($art.summary)"
    }
} catch {
    Write-Error "News test failed: $_"
}

Write-Host "`n--- 3. Testing Direct Batch Scraping ---" -ForegroundColor Cyan
try {
    $scrapeBody = @{
        urls = @("https://news.ycombinator.com")
        concurrency = 2
    } | ConvertTo-Json
    $scraped = Invoke-RestMethod -Uri "$BaseUrl/api/v1/scrape" -Method Post -Body $scrapeBody -ContentType "application/json"
    Write-Host "Scraped $($scraped.count) URLs:" -ForegroundColor Green
    foreach ($res in $scraped.results) {
        Write-Host "  * URL: $($res.url)" -ForegroundColor Yellow
        Write-Host "    Title: $($res.title) (Status: $($res.status_code), Duration: $($res.duration_ms)ms)"
        Write-Host "    Clean Text Preview: $($res.clean_text.Substring(0, [Math]::Min(120, $res.clean_text.Length)))..."
    }
} catch {
    Write-Error "Scrape test failed: $_"
}

Write-Host "`n--- 4. Testing AI Internet Query ---" -ForegroundColor Cyan
try {
    $queryBody = @{
        query = "What are the latest open source AI models?"
        max_results = 3
    } | ConvertTo-Json
    $queryRes = Invoke-RestMethod -Uri "$BaseUrl/api/v1/query" -Method Post -Body $queryBody -ContentType "application/json"
    Write-Host "Query: $($queryRes.query)" -ForegroundColor Green
    Write-Host "Confidence: $($queryRes.confidence) (Time: $($queryRes.duration_ms)ms)"
    Write-Host "Answer:`n$($queryRes.answer)" -ForegroundColor White
} catch {
    Write-Error "Query test failed: $_"
}
