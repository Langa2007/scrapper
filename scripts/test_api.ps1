$BaseUrl = "http://localhost:8080"

Write-Host "Testing health endpoint..."
try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/api/v1/health" -Method Get
    Write-Host "Status:" $health.status "(python:" $health.python_service ")"
} catch {
    Write-Error "Health check failed: $_"
}

Write-Host "Testing news endpoint..."
try {
    $newsBody = @{ topic = "technology"; limit = 2; timeframe = "w" } | ConvertTo-Json
    $news = Invoke-RestMethod -Uri "$BaseUrl/api/v1/news" -Method Post -Body $newsBody -ContentType "application/json"
    Write-Host "Articles received:" $news.count
} catch {
    Write-Error "News test failed: $_"
}

Write-Host "Testing crypto endpoint..."
try {
    $cryptoBody = @{ coin = "bitcoin" } | ConvertTo-Json
    $crypto = Invoke-RestMethod -Uri "$BaseUrl/api/v1/crypto" -Method Post -Body $cryptoBody -ContentType "application/json"
    Write-Host "Coin:" $crypto.name "Price:" $crypto.price_usd "Signal:" $crypto.signal
} catch {
    Write-Error "Crypto test failed: $_"
}

Write-Host "Testing chat endpoint..."
try {
    $chatBody = @{ message = "hello"; mode = "general" } | ConvertTo-Json
    $chat = Invoke-RestMethod -Uri "$BaseUrl/api/v1/chat" -Method Post -Body $chatBody -ContentType "application/json"
    Write-Host "Chat reply:" $chat.reply
} catch {
    Write-Error "Chat test failed: $_"
}

Write-Host "Testing scrape endpoint..."
try {
    $scrapeBody = @{ urls = @("https://news.ycombinator.com"); concurrency = 2 } | ConvertTo-Json
    $scraped = Invoke-RestMethod -Uri "$BaseUrl/api/v1/scrape" -Method Post -Body $scrapeBody -ContentType "application/json"
    Write-Host "Scraped URLs:" $scraped.count
} catch {
    Write-Error "Scrape test failed: $_"
}
